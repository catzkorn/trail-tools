// Package trips stores trip reports, their photos and the public profiles
// they are published under.
package trips

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/trips/internal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	// ErrHandleTaken is returned when another user has the handle.
	ErrHandleTaken = errors.New("the handle is taken")
	// ErrSlugTaken is returned when another of the user's reports has the
	// slug.
	ErrSlugTaken = errors.New("the slug is taken")
	// ErrPhotosMismatch is returned when reordering photos with a list that
	// isn't exactly the report's photos.
	ErrPhotosMismatch = errors.New("the photos don't match the report's photos")
	// ErrPhotoImported is returned when adding a photo from Strava that the
	// report already has.
	ErrPhotoImported = errors.New("the photo was already imported")
)

// Repository stores trip reports, photos and public profiles.
type Repository struct {
	log *slog.Logger
	db  *store.DB
	q   *internal.Queries
}

// NewRepository creates a new Repository from the provided logger and database.
func NewRepository(log *slog.Logger, db *store.DB) (*Repository, error) {
	return &Repository{
		log: log,
		db:  db,
		q:   internal.New(db),
	}, nil
}

// Profile is a user's public profile.
type Profile struct {
	UserID      pgtype.UUID
	Handle      string
	DisplayName string
}

// Report is a trip report.
type Report struct {
	ID         pgtype.UUID
	UserID     pgtype.UUID
	ActivityID pgtype.UUID
	Title      string
	Slug       string
	Body       string
	// The local date of the trip.
	Date      time.Time
	Timezone  string
	SportType string
	// The start time of the activity, zero if unknown.
	StartTime           time.Time
	DistanceMeters      float64
	MovingTimeSeconds   int32
	ElapsedTimeSeconds  int32
	ElevationGainMeters float64
	// A protobuf encoded activities.v1.ActivityStreams message. Nil if the
	// report has no track, or it wasn't loaded.
	Track    []byte
	HasTrack bool
	// Whether the report is published, and when it was first published.
	Published   bool
	PublishTime time.Time
	CreateTime  time.Time
	UpdateTime  time.Time
}

// LocationSource is where the location of a photo comes from.
type LocationSource string

const (
	LocationGPS       LocationSource = "gps"
	LocationTrackTime LocationSource = "track_time"
	LocationManual    LocationSource = "manual"
)

// Location is where a photo was taken.
type Location struct {
	Lat, Lng float64
	Source   LocationSource
	// The distance along the report's track of the point nearest to the
	// location, if the report has a track.
	TrackDistanceMeters    float64
	HasTrackDistanceMeters bool
}

// Photo is a photo of a trip report.
type Photo struct {
	ID       pgtype.UUID
	ReportID pgtype.UUID
	Position int32
	// The images are stored under <ObjectKey>/<size>.jpg.
	ObjectKey string
	Width     int32
	Height    int32
	Caption   string
	// Nil if the photo has no location.
	Location *Location
	// Zero if unknown.
	TakenTime time.Time
	// Strava's ID of the photo, if it was imported from Strava.
	StravaPhotoID string
	CreateTime    time.Time
}

// GetProfile gets the user's public profile.
func (r *Repository) GetProfile(ctx context.Context, userID pgtype.UUID) (*Profile, error) {
	row, err := r.q.GetProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return &Profile{UserID: row.UserID, Handle: row.Handle, DisplayName: row.DisplayName}, nil
}

// GetProfileByHandle gets a public profile by its handle.
func (r *Repository) GetProfileByHandle(ctx context.Context, handle string) (*Profile, error) {
	row, err := r.q.GetProfileByHandle(ctx, handle)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return &Profile{UserID: row.UserID, Handle: row.Handle, DisplayName: row.DisplayName}, nil
}

// UpsertProfile creates or updates the user's public profile. It returns
// ErrHandleTaken if another user has the handle.
func (r *Repository) UpsertProfile(ctx context.Context, p *Profile) (*Profile, error) {
	row, err := r.q.UpsertProfile(ctx, &internal.UpsertProfileParams{
		UserID:      p.UserID,
		Handle:      p.Handle,
		DisplayName: p.DisplayName,
	})
	if err != nil {
		if isUniqueViolation(err, "public_profiles_handle_key") {
			return nil, ErrHandleTaken
		}
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}
	return &Profile{UserID: row.UserID, Handle: row.Handle, DisplayName: row.DisplayName}, nil
}

// GetUserName returns the name the user signed up with.
func (r *Repository) GetUserName(ctx context.Context, userID pgtype.UUID) (string, error) {
	name, err := r.q.GetUserName(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", store.ErrNotFound
		}
		return "", fmt.Errorf("failed to get user name: %w", err)
	}
	return name, nil
}

// CreateReport creates a trip report. It returns ErrSlugTaken if another of
// the user's reports has the slug.
func (r *Repository) CreateReport(ctx context.Context, report *Report) (*Report, error) {
	row, err := r.q.CreateReport(ctx, &internal.CreateReportParams{
		UserID:              report.UserID,
		ActivityID:          report.ActivityID,
		Title:               report.Title,
		Slug:                report.Slug,
		Body:                report.Body,
		Date:                store.DateToPG(report.Date),
		Timezone:            report.Timezone,
		SportType:           report.SportType,
		StartTime:           timestampToPG(report.StartTime),
		DistanceMeters:      report.DistanceMeters,
		MovingTimeSeconds:   report.MovingTimeSeconds,
		ElapsedTimeSeconds:  report.ElapsedTimeSeconds,
		ElevationGainMeters: report.ElevationGainMeters,
		Track:               report.Track,
	})
	if err != nil {
		if isUniqueViolation(err, "trip_reports_user_id_slug_key") {
			return nil, ErrSlugTaken
		}
		return nil, fmt.Errorf("failed to create trip report: %w", err)
	}
	return reportFromDB(row), nil
}

// GetReport gets one of the user's reports, with its track.
func (r *Repository) GetReport(ctx context.Context, userID, reportID pgtype.UUID) (*Report, error) {
	row, err := r.q.GetReport(ctx, &internal.GetReportParams{ID: reportID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get trip report: %w", err)
	}
	return reportFromDB(row), nil
}

// GetReportBySlug gets one of the user's reports by its slug, with its
// track.
func (r *Repository) GetReportBySlug(ctx context.Context, userID pgtype.UUID, slug string) (*Report, error) {
	row, err := r.q.GetReportBySlug(ctx, &internal.GetReportBySlugParams{UserID: userID, Slug: slug})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get trip report: %w", err)
	}
	return reportFromDB(row), nil
}

// ListReportsFilter filters listed reports.
type ListReportsFilter struct {
	// Only list reports made from this activity, if valid.
	ActivityID pgtype.UUID
	// Only list published reports.
	PublishedOnly bool
}

// ListReports lists the user's reports without their tracks, most recent
// trip first.
func (r *Repository) ListReports(ctx context.Context, userID pgtype.UUID, filter ListReportsFilter) ([]*Report, error) {
	rows, err := r.q.ListReports(ctx, &internal.ListReportsParams{
		UserID:        userID,
		ActivityID:    filter.ActivityID,
		PublishedOnly: filter.PublishedOnly,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list trip reports: %w", err)
	}
	reports := make([]*Report, len(rows))
	for i, row := range rows {
		reports[i] = &Report{
			ID:                  row.ID,
			UserID:              row.UserID,
			ActivityID:          row.ActivityID,
			Title:               row.Title,
			Slug:                row.Slug,
			Body:                row.Body,
			Date:                store.DateFromPG(row.Date),
			Timezone:            row.Timezone,
			SportType:           row.SportType,
			StartTime:           timestampFromPG(row.StartTime),
			DistanceMeters:      row.DistanceMeters,
			MovingTimeSeconds:   row.MovingTimeSeconds,
			ElapsedTimeSeconds:  row.ElapsedTimeSeconds,
			ElevationGainMeters: row.ElevationGainMeters,
			HasTrack:            row.HasTrack,
			Published:           row.Published,
			PublishTime:         timestampFromPG(row.PublishTime),
			CreateTime:          row.CreateTime.Time,
			UpdateTime:          row.UpdateTime.Time,
		}
	}
	return reports, nil
}

// ReportUpdate is an update of a report. Nil fields are unchanged.
type ReportUpdate struct {
	ID, UserID  pgtype.UUID
	Title, Slug *string
	Body        *string
	Date        *time.Time
}

// UpdateReport updates one of the user's reports. It returns ErrSlugTaken if
// another of the user's reports has the slug.
func (r *Repository) UpdateReport(ctx context.Context, u *ReportUpdate) (*Report, error) {
	params := &internal.UpdateReportParams{ID: u.ID, UserID: u.UserID}
	if u.Title != nil {
		params.Title = pgtype.Text{String: *u.Title, Valid: true}
	}
	if u.Slug != nil {
		params.Slug = pgtype.Text{String: *u.Slug, Valid: true}
	}
	if u.Body != nil {
		params.Body = pgtype.Text{String: *u.Body, Valid: true}
	}
	if u.Date != nil {
		params.Date = store.DateToPG(*u.Date)
	}
	row, err := r.q.UpdateReport(ctx, params)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, store.ErrNotFound
		case isUniqueViolation(err, "trip_reports_user_id_slug_key"):
			return nil, ErrSlugTaken
		}
		return nil, fmt.Errorf("failed to update trip report: %w", err)
	}
	return reportFromDB(row), nil
}

// SetReportPublished publishes or unpublishes one of the user's reports. The
// publish time is set when it is first published.
func (r *Repository) SetReportPublished(ctx context.Context, userID, reportID pgtype.UUID, published bool) (*Report, error) {
	row, err := r.q.SetReportPublished(ctx, &internal.SetReportPublishedParams{
		ID:        reportID,
		UserID:    userID,
		Published: published,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to publish trip report: %w", err)
	}
	return reportFromDB(row), nil
}

// DeleteReport deletes one of the user's reports and its photos. It returns
// the object keys of the deleted photos, whose images should be deleted.
func (r *Repository) DeleteReport(ctx context.Context, userID, reportID pgtype.UUID) ([]string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())
	q := r.q.WithTx(tx)
	exists, err := q.ReportExists(ctx, &internal.ReportExistsParams{ID: reportID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get trip report: %w", err)
	}
	if !exists {
		return nil, store.ErrNotFound
	}
	keys, err := q.ListPhotoObjectKeys(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("failed to list photos: %w", err)
	}
	if _, err := q.DeleteReport(ctx, &internal.DeleteReportParams{ID: reportID, UserID: userID}); err != nil {
		return nil, fmt.Errorf("failed to delete trip report: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return keys, nil
}

// AddPhoto adds a photo to the end of one of the user's reports.
func (r *Repository) AddPhoto(ctx context.Context, userID pgtype.UUID, p *Photo) (*Photo, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())
	q := r.q.WithTx(tx)
	exists, err := q.ReportExists(ctx, &internal.ReportExistsParams{ID: p.ReportID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get trip report: %w", err)
	}
	if !exists {
		return nil, store.ErrNotFound
	}
	params := &internal.AddPhotoParams{
		ReportID:      p.ReportID,
		ObjectKey:     p.ObjectKey,
		Width:         p.Width,
		Height:        p.Height,
		Caption:       p.Caption,
		TakenTime:     timestampToPG(p.TakenTime),
		StravaPhotoID: pgtype.Text{String: p.StravaPhotoID, Valid: p.StravaPhotoID != ""},
	}
	params.Lat, params.Lng, params.LocationSource, params.TrackDistanceMeters = locationToPG(p.Location)
	row, err := q.AddPhoto(ctx, params)
	if err != nil {
		if isUniqueViolation(err, "trip_photos_report_id_strava_photo_id_key") {
			return nil, ErrPhotoImported
		}
		return nil, fmt.Errorf("failed to add photo: %w", err)
	}
	if err := q.TouchReport(ctx, p.ReportID); err != nil {
		return nil, fmt.Errorf("failed to update trip report: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return photoFromDB(row), nil
}

// ListPhotos lists the photos of reports in order. The caller must check
// that the reports can be accessed.
func (r *Repository) ListPhotos(ctx context.Context, reportIDs ...pgtype.UUID) (map[pgtype.UUID][]*Photo, error) {
	rows, err := r.q.ListPhotos(ctx, reportIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list photos: %w", err)
	}
	photos := map[pgtype.UUID][]*Photo{}
	for _, row := range rows {
		photos[row.ReportID] = append(photos[row.ReportID], photoFromDB(row))
	}
	return photos, nil
}

// GetPhoto gets a photo of one of the user's reports.
func (r *Repository) GetPhoto(ctx context.Context, userID, photoID pgtype.UUID) (*Photo, error) {
	row, err := r.q.GetPhoto(ctx, &internal.GetPhotoParams{ID: photoID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get photo: %w", err)
	}
	return photoFromDB(row), nil
}

// PhotoAccess is what's needed to serve a photo: where it's stored, and who
// can see it.
type PhotoAccess struct {
	ObjectKey  string
	CreateTime time.Time
	// The author of the report.
	UserID pgtype.UUID
	// Whether the report is published.
	Published bool
}

// GetPhotoAccess gets what's needed to serve any photo.
func (r *Repository) GetPhotoAccess(ctx context.Context, photoID pgtype.UUID) (*PhotoAccess, error) {
	row, err := r.q.GetPhotoAccess(ctx, photoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get photo: %w", err)
	}
	return &PhotoAccess{
		ObjectKey:  row.ObjectKey,
		CreateTime: row.CreateTime.Time,
		UserID:     row.UserID,
		Published:  row.Published,
	}, nil
}

// UpdatePhotoCaption sets the caption of a photo of one of the user's
// reports.
func (r *Repository) UpdatePhotoCaption(ctx context.Context, userID, photoID pgtype.UUID, caption string) (*Photo, error) {
	row, err := r.q.UpdatePhotoCaption(ctx, &internal.UpdatePhotoCaptionParams{ID: photoID, UserID: userID, Caption: caption})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to update photo: %w", err)
	}
	return photoFromDB(row), nil
}

// UpdatePhotoLocation sets or, with a nil location, clears the location of
// a photo of one of the user's reports.
func (r *Repository) UpdatePhotoLocation(ctx context.Context, userID, photoID pgtype.UUID, loc *Location) (*Photo, error) {
	params := &internal.UpdatePhotoLocationParams{ID: photoID, UserID: userID}
	params.Lat, params.Lng, params.LocationSource, params.TrackDistanceMeters = locationToPG(loc)
	row, err := r.q.UpdatePhotoLocation(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to update photo: %w", err)
	}
	return photoFromDB(row), nil
}

// ReorderPhotos orders the photos of one of the user's reports. photoIDs
// must be exactly the report's photos, or ErrPhotosMismatch is returned.
func (r *Repository) ReorderPhotos(ctx context.Context, userID, reportID pgtype.UUID, photoIDs []pgtype.UUID) ([]*Photo, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())
	q := r.q.WithTx(tx)
	exists, err := q.ReportExists(ctx, &internal.ReportExistsParams{ID: reportID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get trip report: %w", err)
	}
	if !exists {
		return nil, store.ErrNotFound
	}
	current, err := q.ListPhotos(ctx, []pgtype.UUID{reportID})
	if err != nil {
		return nil, fmt.Errorf("failed to list photos: %w", err)
	}
	remaining := map[pgtype.UUID]bool{}
	for _, p := range current {
		remaining[p.ID] = true
	}
	for _, id := range photoIDs {
		if !remaining[id] {
			return nil, ErrPhotosMismatch
		}
		delete(remaining, id)
	}
	if len(remaining) > 0 {
		return nil, ErrPhotosMismatch
	}
	if err := q.SetPhotoPositions(ctx, &internal.SetPhotoPositionsParams{ReportID: reportID, PhotoIds: photoIDs}); err != nil {
		return nil, fmt.Errorf("failed to reorder photos: %w", err)
	}
	if err := q.TouchReport(ctx, reportID); err != nil {
		return nil, fmt.Errorf("failed to update trip report: %w", err)
	}
	rows, err := q.ListPhotos(ctx, []pgtype.UUID{reportID})
	if err != nil {
		return nil, fmt.Errorf("failed to list photos: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	photos := make([]*Photo, len(rows))
	for i, row := range rows {
		photos[i] = photoFromDB(row)
	}
	return photos, nil
}

// DeletePhoto deletes a photo of one of the user's reports. It returns the
// photo's object key, whose images should be deleted.
func (r *Repository) DeletePhoto(ctx context.Context, userID, photoID pgtype.UUID) (string, error) {
	key, err := r.q.DeletePhoto(ctx, &internal.DeletePhotoParams{ID: photoID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", store.ErrNotFound
		}
		return "", fmt.Errorf("failed to delete photo: %w", err)
	}
	return key, nil
}

func reportFromDB(row *internal.TripReport) *Report {
	return &Report{
		ID:                  row.ID,
		UserID:              row.UserID,
		ActivityID:          row.ActivityID,
		Title:               row.Title,
		Slug:                row.Slug,
		Body:                row.Body,
		Date:                store.DateFromPG(row.Date),
		Timezone:            row.Timezone,
		SportType:           row.SportType,
		StartTime:           timestampFromPG(row.StartTime),
		DistanceMeters:      row.DistanceMeters,
		MovingTimeSeconds:   row.MovingTimeSeconds,
		ElapsedTimeSeconds:  row.ElapsedTimeSeconds,
		ElevationGainMeters: row.ElevationGainMeters,
		Track:               row.Track,
		HasTrack:            row.Track != nil,
		Published:           row.Published,
		PublishTime:         timestampFromPG(row.PublishTime),
		CreateTime:          row.CreateTime.Time,
		UpdateTime:          row.UpdateTime.Time,
	}
}

func photoFromDB(row *internal.TripPhoto) *Photo {
	p := &Photo{
		ID:            row.ID,
		ReportID:      row.ReportID,
		Position:      row.Position,
		ObjectKey:     row.ObjectKey,
		Width:         row.Width,
		Height:        row.Height,
		Caption:       row.Caption,
		TakenTime:     timestampFromPG(row.TakenTime),
		StravaPhotoID: row.StravaPhotoID.String,
		CreateTime:    row.CreateTime.Time,
	}
	if row.Lat.Valid && row.Lng.Valid && row.LocationSource.Valid {
		p.Location = &Location{
			Lat:                    row.Lat.Float64,
			Lng:                    row.Lng.Float64,
			Source:                 LocationSource(row.LocationSource.PhotoLocationSource),
			TrackDistanceMeters:    row.TrackDistanceMeters.Float64,
			HasTrackDistanceMeters: row.TrackDistanceMeters.Valid,
		}
	}
	return p
}

func locationToPG(loc *Location) (lat, lng pgtype.Float8, source internal.NullPhotoLocationSource, trackDistance pgtype.Float8) {
	if loc == nil {
		return lat, lng, source, trackDistance
	}
	lat = pgtype.Float8{Float64: loc.Lat, Valid: true}
	lng = pgtype.Float8{Float64: loc.Lng, Valid: true}
	source = internal.NullPhotoLocationSource{PhotoLocationSource: internal.PhotoLocationSource(loc.Source), Valid: true}
	if loc.HasTrackDistanceMeters {
		trackDistance = pgtype.Float8{Float64: loc.TrackDistanceMeters, Valid: true}
	}
	return lat, lng, source, trackDistance
}

func timestampToPG(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func timestampFromPG(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

// isUniqueViolation returns whether err is a violation of the unique
// constraint.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
