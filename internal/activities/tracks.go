package activities

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/catzkorn/trail-tools/internal/activities/internal/gen"
	activitiesv1 "github.com/catzkorn/trail-tools/internal/gen/activities/v1"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"
)

const (
	// TrackToleranceMeters is how far the stored tracks may stray from the
	// recorded ones.
	TrackToleranceMeters = 2
	// SummaryPrecisionMeters is the precision from which ListTracksInBounds
	// lists the summary polylines of the activities, which Strava simplifies
	// to tens of meters, instead of their tracks.
	SummaryPrecisionMeters = 50
	// backfillBatchSize is how many streams are read at once to compute the
	// tracks missing from them.
	backfillBatchSize = 20
)

// Track is the GPS track of an activity.
type Track struct {
	ActivityID pgtype.UUID
	// The points of the track, without elevations.
	Points []geo.Point
}

// ListTracksInBounds lists the tracks of activities whose bounding box
// intersects the bounds, optionally within an inclusive date range.
// precision is how far in meters the tracks may stray from the recorded
// ones: they are the tracks from the streams, simplified to
// TrackToleranceMeters, or for a precision of SummaryPrecisionMeters or more
// the much smaller summary polylines. Activities without a track, such as
// manual ones, are left out.
func (r *Repository) ListTracksInBounds(ctx context.Context, userID pgtype.UUID, bounds geo.Bounds, startDate, endDate time.Time, precision float64) ([]Track, error) {
	if precision >= SummaryPrecisionMeters {
		rows, err := r.q.ListActivitySummaryTracksInBounds(ctx, &gen.ListActivitySummaryTracksInBoundsParams{
			UserID:    userID,
			North:     bounds.MaxLat,
			South:     bounds.MinLat,
			East:      bounds.MaxLng,
			West:      bounds.MinLng,
			StartDate: store.DateToPG(startDate),
			EndDate:   store.DateToPG(endDate),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list tracks: %w", err)
		}
		tracks := make([]Track, 0, len(rows))
		for _, row := range rows {
			tracks = r.appendTrack(ctx, tracks, row.ID, row.SummaryPolyline)
		}
		return tracks, nil
	}
	rows, err := r.q.ListActivityTracksInBounds(ctx, &gen.ListActivityTracksInBoundsParams{
		UserID:    userID,
		North:     bounds.MaxLat,
		South:     bounds.MinLat,
		East:      bounds.MaxLng,
		West:      bounds.MinLng,
		StartDate: store.DateToPG(startDate),
		EndDate:   store.DateToPG(endDate),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tracks: %w", err)
	}
	var missing []pgtype.UUID
	for _, row := range rows {
		if row.HasStreams && !row.TrackPolyline.Valid {
			missing = append(missing, row.ID)
		}
	}
	backfilled, err := r.backfillTracks(ctx, missing)
	if err != nil {
		return nil, err
	}
	tracks := make([]Track, 0, len(rows))
	for _, row := range rows {
		polyline := row.TrackPolyline.String
		if row.HasStreams && !row.TrackPolyline.Valid {
			polyline = backfilled[row.ID.Bytes]
		}
		// Streams without positions, or none at all.
		if polyline == "" {
			polyline = row.SummaryPolyline
		}
		tracks = r.appendTrack(ctx, tracks, row.ID, polyline)
	}
	return tracks, nil
}

// appendTrack appends the track of an encoded polyline to tracks, unless it's
// empty or can't be decoded.
func (r *Repository) appendTrack(ctx context.Context, tracks []Track, activityID pgtype.UUID, polyline string) []Track {
	if polyline == "" {
		return tracks
	}
	points, err := geo.DecodePolyline(polyline)
	if err != nil {
		r.report.Report(ctx, "Skipping undecodable track", err, slog.String("activity_id", store.UUIDToString(activityID)))
		return tracks
	}
	return append(tracks, Track{ActivityID: activityID, Points: points})
}

// backfillTracks computes and stores the tracks of activities whose streams
// were synced before tracks were stored with them. It returns the encoded
// tracks by activity ID.
func (r *Repository) backfillTracks(ctx context.Context, activityIDs []pgtype.UUID) (map[[16]byte]string, error) {
	tracks := make(map[[16]byte]string, len(activityIDs))
	for batch := range slices.Chunk(activityIDs, backfillBatchSize) {
		rows, err := r.q.ListStreamsWithoutTrack(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to list streams without a track: %w", err)
		}
		for _, row := range rows {
			track, err := trackFromStreams(row.Streams)
			if err != nil {
				// Stored as empty, so they aren't decoded again, and the
				// summary polyline is used instead.
				r.report.Report(ctx, "Undecodable streams", err, slog.String("activity_id", store.UUIDToString(row.ActivityID)))
			}
			if err := r.q.SetActivityTrack(ctx, &gen.SetActivityTrackParams{ActivityID: row.ActivityID, TrackPolyline: track}); err != nil {
				return nil, fmt.Errorf("failed to store track: %w", err)
			}
			tracks[row.ActivityID.Bytes] = track
		}
	}
	return tracks, nil
}

// trackFromStreams returns the track of encoded streams, simplified to
// TrackToleranceMeters and encoded as a polyline. It's empty if the streams
// have no positions.
func trackFromStreams(data []byte) (string, error) {
	var streams activitiesv1.ActivityStreams
	if err := proto.Unmarshal(data, &streams); err != nil {
		return "", fmt.Errorf("failed to decode streams: %w", err)
	}
	lats, lngs := streams.GetLat(), streams.GetLng()
	points := make([]geo.Point, min(len(lats), len(lngs)))
	for i := range points {
		points[i] = geo.Point{Lat: lats[i], Lng: lngs[i]}
	}
	return geo.EncodePolyline(geo.Simplify(points, TrackToleranceMeters)), nil
}
