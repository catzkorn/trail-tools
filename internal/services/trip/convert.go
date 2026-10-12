package trip

import (
	geov1 "github.com/catzkorn/trail-tools/internal/gen/geo/v1"
	tripsv1 "github.com/catzkorn/trail-tools/internal/gen/trips/v1"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/trips"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// reportToProto converts a report with its photos. profile is the author's
// profile, nil if the author has none.
func (s *Service) reportToProto(r *trips.Report, photos []*trips.Photo, profile *trips.Profile) *tripsv1.TripReport {
	stats := tripsv1.TripStats_builder{
		SportType:           r.SportType,
		Timezone:            r.Timezone,
		DistanceMeters:      r.DistanceMeters,
		MovingTimeSeconds:   r.MovingTimeSeconds,
		ElapsedTimeSeconds:  r.ElapsedTimeSeconds,
		ElevationGainMeters: r.ElevationGainMeters,
	}
	if !r.StartTime.IsZero() {
		stats.StartTime = timestamppb.New(r.StartTime)
	}
	b := tripsv1.TripReport_builder{
		Id:               store.UUIDToString(r.ID),
		ActivityId:       store.UUIDToString(r.ActivityID),
		Title:            r.Title,
		Slug:             r.Slug,
		Body:             r.Body,
		Date:             store.FormatDate(r.Date),
		Stats:            stats.Build(),
		Published:        r.Published,
		HasTrack:         r.HasTrack,
		HiddenEndsMeters: r.HiddenEndsMeters,
		CreateTime:       timestamppb.New(r.CreateTime),
		UpdateTime:       timestamppb.New(r.UpdateTime),
	}
	if !r.PublishTime.IsZero() {
		b.PublishTime = timestamppb.New(r.PublishTime)
	}
	if profile != nil {
		b.Url = s.reportURL(profile.Handle, r.Slug)
	}
	for _, p := range photos {
		b.Photos = append(b.Photos, s.photoToProto(p))
	}
	return b.Build()
}

func (s *Service) photoToProto(p *trips.Photo) *tripsv1.TripPhoto {
	id := store.UUIDToString(p.ID)
	b := tripsv1.TripPhoto_builder{
		Id:           id,
		Caption:      p.Caption,
		Width:        p.Width,
		Height:       p.Height,
		Url:          s.photoURL(id, sizeLarge),
		ThumbnailUrl: s.photoURL(id, sizeThumb),
	}
	if loc := p.Location; loc != nil {
		b.Location = geov1.LatLng_builder{Lat: loc.Lat, Lng: loc.Lng}.Build()
		b.LocationSource = locationSourceToProto[loc.Source]
		if loc.HasTrackDistanceMeters {
			b.TrackDistanceMeters = &loc.TrackDistanceMeters
		}
	}
	if !p.TakenTime.IsZero() {
		b.TakenTime = timestamppb.New(p.TakenTime)
	}
	return b.Build()
}

var locationSourceToProto = map[trips.LocationSource]tripsv1.LocationSource{
	trips.LocationGPS:       tripsv1.LocationSource_LOCATION_SOURCE_GPS,
	trips.LocationTrackTime: tripsv1.LocationSource_LOCATION_SOURCE_TRACK_TIME,
	trips.LocationManual:    tripsv1.LocationSource_LOCATION_SOURCE_MANUAL,
}

func (s *Service) profileToProto(p *trips.Profile) *tripsv1.Profile {
	return tripsv1.Profile_builder{
		Handle:      p.Handle,
		DisplayName: p.DisplayName,
		Url:         s.profileURL(p.Handle),
	}.Build()
}

func (s *Service) profileURL(handle string) string {
	return s.baseURL + ProfilePath + handle
}

func (s *Service) reportURL(handle, slug string) string {
	return s.baseURL + ProfilePath + handle + "/" + slug
}

func (s *Service) photoURL(id, size string) string {
	return s.baseURL + PhotoPath + id + "/" + size + ".jpg"
}
