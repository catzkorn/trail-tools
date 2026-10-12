package services_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"github.com/catzkorn/trail-tools/internal/activities"
	activitiesv1 "github.com/catzkorn/trail-tools/internal/gen/activities/v1"
	geov1 "github.com/catzkorn/trail-tools/internal/gen/geo/v1"
	tripsv1 "github.com/catzkorn/trail-tools/internal/gen/trips/v1"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/store"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// tripStart is when the test activity starts: 10:00 in Zurich.
var tripStart = time.Date(2026, 9, 7, 8, 0, 0, 0, time.UTC)

// createActivity stores an activity of the user with streams: 101 points
// 36 seconds and 100 meters apart, heading north from the equator.
func (e *env) createActivity() string {
	e.t.Helper()
	ctx := context.Background()
	points := []geo.Point{{Lat: 0, Lng: 0}, {Lat: 0.09, Lng: 0}}
	id, err := e.activities.UpsertActivity(ctx, e.userID, &activities.ActivitySummary{
		StravaID:            4242,
		Name:                "Ridge run",
		SportType:           "TrailRun",
		StartTime:           tripStart,
		StartDateLocal:      time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC),
		Timezone:            "Europe/Zurich",
		DistanceMeters:      10000,
		MovingTimeSeconds:   3500,
		ElapsedTimeSeconds:  3600,
		ElevationGainMeters: 500,
		SummaryPolyline:     geo.EncodePolyline(points),
		RawJSON:             []byte("{}"),
	})
	if err != nil {
		e.t.Fatal(err)
	}
	if err := e.activities.SetActivityDetail(ctx, id, "Felt good", "Watch", []byte("{}")); err != nil {
		e.t.Fatal(err)
	}
	streams := &activitiesv1.ActivityStreams{}
	for i := range 101 {
		streams.SetTimeSeconds(append(streams.GetTimeSeconds(), int32(i*36)))
		streams.SetDistanceMeters(append(streams.GetDistanceMeters(), float64(i)*100))
		streams.SetLat(append(streams.GetLat(), float64(i)*0.0009))
		streams.SetLng(append(streams.GetLng(), 0))
		streams.SetAltitudeMeters(append(streams.GetAltitudeMeters(), float64(i)*5))
		streams.SetHeartrateBpm(append(streams.GetHeartrateBpm(), 150))
	}
	data, err := proto.Marshal(streams)
	if err != nil {
		e.t.Fatal(err)
	}
	if err := e.activities.SetActivityStreams(ctx, id, data, &geo.Bounds{MaxLat: 0.09}); err != nil {
		e.t.Fatal(err)
	}
	return store.UUIDToString(id)
}

// createProfile gives the user a public profile.
func (e *env) createProfile(handle string) {
	e.t.Helper()
	if _, err := e.trips.UpdateProfile(context.Background(), tripsv1.UpdateProfileRequest_builder{
		Profile: tripsv1.Profile_builder{Handle: handle, DisplayName: "Jane Doe"}.Build(),
	}.Build()); err != nil {
		e.t.Fatal(err)
	}
}

// get makes a GET request to the server, authenticated with the token if
// it's not empty, and returns the response with its body read.
func (e *env) get(path, token string) (*http.Response, []byte) {
	e.t.Helper()
	req, err := http.NewRequest(http.MethodGet, e.srv.URL+strings.TrimPrefix(path, "https://trail.example"), nil)
	if err != nil {
		e.t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := e.srv.Client().Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatal(err)
	}
	return resp, body
}

func TestTripReports(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()
	activityID := e.createActivity()

	_, err := e.anonTrips.CreateTripReport(ctx, tripsv1.CreateTripReportRequest_builder{ActivityId: activityID}.Build())
	wantCode(t, err, connect.CodeUnauthenticated)
	_, err = e.otherTrips.CreateTripReport(ctx, tripsv1.CreateTripReportRequest_builder{ActivityId: activityID}.Build())
	wantCode(t, err, connect.CodeNotFound)

	// A report made from an activity is prefilled from it.
	created, err := e.trips.CreateTripReport(ctx, tripsv1.CreateTripReportRequest_builder{ActivityId: activityID}.Build())
	if err != nil {
		t.Fatalf("CreateTripReport: %v", err)
	}
	report := created.GetReport()
	stats := report.GetStats()
	if report.GetTitle() != "Ridge run" || report.GetSlug() != "ridge-run" || report.GetDate() != "2026-09-07" ||
		report.GetBody() != "Felt good" || report.GetActivityId() != activityID || !report.GetHasTrack() ||
		report.GetPublished() || report.GetUrl() != "" || report.HasPublishTime() {
		t.Fatalf("unexpected report %v", report)
	}
	if stats.GetSportType() != "TrailRun" || stats.GetDistanceMeters() != 10000 || stats.GetElevationGainMeters() != 500 ||
		stats.GetMovingTimeSeconds() != 3500 || stats.GetElapsedTimeSeconds() != 3600 ||
		stats.GetTimezone() != "Europe/Zurich" || !stats.GetStartTime().AsTime().Equal(tripStart) {
		t.Errorf("unexpected stats %v", stats)
	}

	// Slugs are unique per author.
	second, err := e.trips.CreateTripReport(ctx, tripsv1.CreateTripReportRequest_builder{ActivityId: activityID}.Build())
	if err != nil {
		t.Fatalf("CreateTripReport: %v", err)
	}
	if second.GetReport().GetSlug() != "ridge-run-2" {
		t.Errorf("second slug = %q, want ridge-run-2", second.GetReport().GetSlug())
	}
	_, err = e.trips.CreateTripReport(ctx, tripsv1.CreateTripReportRequest_builder{}.Build())
	wantCode(t, err, connect.CodeInvalidArgument)
	standalone, err := e.trips.CreateTripReport(ctx, tripsv1.CreateTripReportRequest_builder{Title: "Säntis!"}.Build())
	if err != nil {
		t.Fatalf("CreateTripReport: %v", err)
	}
	if r := standalone.GetReport(); r.GetSlug() != "santis" || r.GetHasTrack() || r.GetActivityId() != "" || r.GetDate() == "" {
		t.Errorf("unexpected report without activity %v", r)
	}
	// Another user can use the same slug.
	otherReport, err := e.otherTrips.CreateTripReport(ctx, tripsv1.CreateTripReportRequest_builder{Title: "Ridge run"}.Build())
	if err != nil || otherReport.GetReport().GetSlug() != "ridge-run" {
		t.Fatalf("CreateTripReport for another user = %v, %v", otherReport, err)
	}

	got, err := e.trips.GetTripReport(ctx, tripsv1.GetTripReportRequest_builder{Id: report.GetId(), IncludeTrack: true}.Build())
	if err != nil {
		t.Fatalf("GetTripReport: %v", err)
	}
	track := got.GetTrack()
	if len(track) != 101 || track[100].GetDistanceMeters() != 10000 || track[100].GetElevationMeters() != 500 {
		t.Errorf("unexpected track of %d points", len(track))
	}
	_, err = e.otherTrips.GetTripReport(ctx, tripsv1.GetTripReportRequest_builder{Id: report.GetId()}.Build())
	wantCode(t, err, connect.CodeNotFound)

	list, err := e.trips.ListTripReports(ctx, tripsv1.ListTripReportsRequest_builder{}.Build())
	if err != nil || len(list.GetReports()) != 3 {
		t.Fatalf("ListTripReports = %v, %v, want 3 reports", list, err)
	}
	byActivity, err := e.trips.ListTripReports(ctx, tripsv1.ListTripReportsRequest_builder{ActivityId: activityID}.Build())
	if err != nil || len(byActivity.GetReports()) != 2 {
		t.Fatalf("ListTripReports by activity = %v, %v, want 2 reports", byActivity, err)
	}

	updated, err := e.trips.UpdateTripReport(ctx, tripsv1.UpdateTripReportRequest_builder{
		Id:    report.GetId(),
		Title: new("Up the ridge"),
		Body:  new("The climb.\n\nThe descent."),
		Date:  new("2026-09-08"),
	}.Build())
	if err != nil {
		t.Fatalf("UpdateTripReport: %v", err)
	}
	if r := updated.GetReport(); r.GetTitle() != "Up the ridge" || r.GetSlug() != "ridge-run" || r.GetBody() != "The climb.\n\nThe descent." || r.GetDate() != "2026-09-08" {
		t.Errorf("unexpected updated report %v", r)
	}
	for _, slug := range []string{"Bad Slug", "", "-x"} {
		_, err = e.trips.UpdateTripReport(ctx, tripsv1.UpdateTripReportRequest_builder{Id: report.GetId(), Slug: new(slug)}.Build())
		wantCode(t, err, connect.CodeInvalidArgument)
	}
	_, err = e.trips.UpdateTripReport(ctx, tripsv1.UpdateTripReportRequest_builder{Id: report.GetId(), Slug: new("ridge-run-2")}.Build())
	wantCode(t, err, connect.CodeAlreadyExists)
	_, err = e.trips.UpdateTripReport(ctx, tripsv1.UpdateTripReportRequest_builder{Id: report.GetId(), Title: new(" ")}.Build())
	wantCode(t, err, connect.CodeInvalidArgument)
	_, err = e.otherTrips.UpdateTripReport(ctx, tripsv1.UpdateTripReportRequest_builder{Id: report.GetId(), Title: new("Mine")}.Build())
	wantCode(t, err, connect.CodeNotFound)

	// Publishing requires a public profile.
	_, err = e.trips.PublishTripReport(ctx, tripsv1.PublishTripReportRequest_builder{Id: report.GetId()}.Build())
	wantCode(t, err, connect.CodeFailedPrecondition)
	profile, err := e.trips.GetProfile(ctx, tripsv1.GetProfileRequest_builder{}.Build())
	if err != nil || profile.HasProfile() || profile.GetSuggestedDisplayName() != "Test user" {
		t.Fatalf("GetProfile = %v, %v", profile, err)
	}
	for _, handle := range []string{"ab", "jane doe", "-jane", "jane_doe"} {
		_, err = e.trips.UpdateProfile(ctx, tripsv1.UpdateProfileRequest_builder{
			Profile: tripsv1.Profile_builder{Handle: handle, DisplayName: "Jane"}.Build(),
		}.Build())
		wantCode(t, err, connect.CodeInvalidArgument)
	}
	_, err = e.trips.UpdateProfile(ctx, tripsv1.UpdateProfileRequest_builder{
		Profile: tripsv1.Profile_builder{Handle: "jane", DisplayName: " "}.Build(),
	}.Build())
	wantCode(t, err, connect.CodeInvalidArgument)
	updatedProfile, err := e.trips.UpdateProfile(ctx, tripsv1.UpdateProfileRequest_builder{
		Profile: tripsv1.Profile_builder{Handle: " Jane-Doe ", DisplayName: "  Jane   Doe "}.Build(),
	}.Build())
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if p := updatedProfile.GetProfile(); p.GetHandle() != "jane-doe" || p.GetDisplayName() != "Jane Doe" || p.GetUrl() != "https://trail.example/u/jane-doe" {
		t.Errorf("unexpected profile %v", p)
	}
	// Handles are unique.
	_, err = e.otherTrips.UpdateProfile(ctx, tripsv1.UpdateProfileRequest_builder{
		Profile: tripsv1.Profile_builder{Handle: "jane-doe", DisplayName: "Not Jane"}.Build(),
	}.Build())
	wantCode(t, err, connect.CodeAlreadyExists)

	// Drafts are only visible to their author.
	publicReq := tripsv1.GetPublicTripReportRequest_builder{Handle: "jane-doe", Slug: "ridge-run"}.Build()
	_, err = e.anonTrips.GetPublicTripReport(ctx, publicReq)
	wantCode(t, err, connect.CodeNotFound)
	_, err = e.otherTrips.GetPublicTripReport(ctx, publicReq)
	wantCode(t, err, connect.CodeNotFound)
	preview, err := e.trips.GetPublicTripReport(ctx, publicReq)
	if err != nil || preview.GetReport().GetPublished() {
		t.Fatalf("GetPublicTripReport as the author = %v, %v", preview, err)
	}
	publicProfile, err := e.anonTrips.GetPublicProfile(ctx, tripsv1.GetPublicProfileRequest_builder{Handle: "jane-doe"}.Build())
	if err != nil || publicProfile.GetProfile().GetDisplayName() != "Jane Doe" || len(publicProfile.GetReports()) != 0 {
		t.Fatalf("GetPublicProfile = %v, %v", publicProfile, err)
	}
	_, err = e.anonTrips.GetPublicProfile(ctx, tripsv1.GetPublicProfileRequest_builder{Handle: "nobody"}.Build())
	wantCode(t, err, connect.CodeNotFound)

	_, err = e.otherTrips.PublishTripReport(ctx, tripsv1.PublishTripReportRequest_builder{Id: report.GetId()}.Build())
	wantCode(t, err, connect.CodeNotFound)
	published, err := e.trips.PublishTripReport(ctx, tripsv1.PublishTripReportRequest_builder{Id: report.GetId()}.Build())
	if err != nil {
		t.Fatalf("PublishTripReport: %v", err)
	}
	if r := published.GetReport(); !r.GetPublished() || !r.HasPublishTime() || r.GetUrl() != "https://trail.example/u/jane-doe/ridge-run" {
		t.Errorf("unexpected published report %v", r)
	}
	public, err := e.anonTrips.GetPublicTripReport(ctx, publicReq)
	if err != nil {
		t.Fatalf("GetPublicTripReport: %v", err)
	}
	if public.GetReport().GetTitle() != "Up the ridge" || public.GetAuthor().GetHandle() != "jane-doe" {
		t.Errorf("unexpected public report %v", public.GetReport())
	}
	// By default the track is hidden within 500 m of its start and finish:
	// of 101 points 100 m apart, the first and last 5.
	if track := public.GetTrack(); len(track) != 91 || track[0].GetDistanceMeters() != 500 {
		t.Errorf("got %d public track points from %v m, want 91 from 500 m", len(track), track[0].GetDistanceMeters())
	}
	// A photo near the start is shown without its place on the map.
	photo, err := e.trips.UploadTripPhoto(ctx, tripsv1.UploadTripPhotoRequest_builder{
		ReportId: report.GetId(),
		Data:     testJPEG(t, 300, 200),
		Location: geov1.LatLng_builder{Lat: 0.001, Lng: 0}.Build(),
	}.Build())
	if err != nil {
		t.Fatalf("UploadTripPhoto: %v", err)
	}
	if public, err = e.anonTrips.GetPublicTripReport(ctx, publicReq); err != nil {
		t.Fatal(err)
	}
	if photos := public.GetReport().GetPhotos(); len(photos) != 1 || photos[0].HasLocation() || photos[0].HasTrackDistanceMeters() {
		t.Errorf("got public photos %v, want the photo near the start without its location", photos)
	}
	// Hiding can be turned off.
	if _, err := e.trips.UpdateTripReport(ctx, tripsv1.UpdateTripReportRequest_builder{Id: report.GetId(), HiddenEndsMeters: new(int32(0))}.Build()); err != nil {
		t.Fatalf("UpdateTripReport: %v", err)
	}
	if public, err = e.anonTrips.GetPublicTripReport(ctx, publicReq); err != nil {
		t.Fatal(err)
	}
	if len(public.GetTrack()) != 101 || !public.GetReport().GetPhotos()[0].HasLocation() {
		t.Errorf("got %d public track points and photos %v, want the whole track and the photo's location", len(public.GetTrack()), public.GetReport().GetPhotos())
	}
	if _, err := e.trips.DeleteTripPhoto(ctx, tripsv1.DeleteTripPhotoRequest_builder{Id: photo.GetPhoto().GetId()}.Build()); err != nil {
		t.Fatal(err)
	}
	_, err = e.trips.UpdateTripReport(ctx, tripsv1.UpdateTripReportRequest_builder{Id: report.GetId(), HiddenEndsMeters: new(int32(5000))}.Build())
	wantCode(t, err, connect.CodeInvalidArgument)
	publicProfile, err = e.anonTrips.GetPublicProfile(ctx, tripsv1.GetPublicProfileRequest_builder{Handle: "jane-doe"}.Build())
	if err != nil || len(publicProfile.GetReports()) != 1 || publicProfile.GetReports()[0].GetSlug() != "ridge-run" {
		t.Fatalf("GetPublicProfile = %v, %v", publicProfile, err)
	}

	// The URL of a published report is stable.
	_, err = e.trips.UpdateTripReport(ctx, tripsv1.UpdateTripReportRequest_builder{Id: report.GetId(), Slug: new("new-slug")}.Build())
	wantCode(t, err, connect.CodeFailedPrecondition)
	if _, err := e.trips.UpdateTripReport(ctx, tripsv1.UpdateTripReportRequest_builder{Id: report.GetId(), Slug: new("ridge-run")}.Build()); err != nil {
		t.Errorf("UpdateTripReport with the same slug: %v", err)
	}

	// Unpublishing makes it a draft again, but keeps the first publish time.
	if _, err := e.trips.UnpublishTripReport(ctx, tripsv1.UnpublishTripReportRequest_builder{Id: report.GetId()}.Build()); err != nil {
		t.Fatalf("UnpublishTripReport: %v", err)
	}
	_, err = e.anonTrips.GetPublicTripReport(ctx, publicReq)
	wantCode(t, err, connect.CodeNotFound)
	if _, err := e.trips.UpdateTripReport(ctx, tripsv1.UpdateTripReportRequest_builder{Id: report.GetId(), Slug: new("new-slug")}.Build()); err != nil {
		t.Errorf("UpdateTripReport of a draft's slug: %v", err)
	}
	republished, err := e.trips.PublishTripReport(ctx, tripsv1.PublishTripReportRequest_builder{Id: report.GetId()}.Build())
	if err != nil {
		t.Fatalf("PublishTripReport: %v", err)
	}
	if !republished.GetReport().GetPublishTime().AsTime().Equal(published.GetReport().GetPublishTime().AsTime()) {
		t.Error("republishing changed the publish time")
	}

	_, err = e.otherTrips.DeleteTripReport(ctx, tripsv1.DeleteTripReportRequest_builder{Id: report.GetId()}.Build())
	wantCode(t, err, connect.CodeNotFound)
	if _, err := e.trips.DeleteTripReport(ctx, tripsv1.DeleteTripReportRequest_builder{Id: report.GetId()}.Build()); err != nil {
		t.Fatalf("DeleteTripReport: %v", err)
	}
	_, err = e.trips.GetTripReport(ctx, tripsv1.GetTripReportRequest_builder{Id: report.GetId()}.Build())
	wantCode(t, err, connect.CodeNotFound)
	_, err = e.anonTrips.ListTripReports(ctx, tripsv1.ListTripReportsRequest_builder{}.Build())
	wantCode(t, err, connect.CodeUnauthenticated)
}

// testJPEG returns a JPEG image of the size.
func testJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 80, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// storedImages returns the number of images in the photo directory.
func (e *env) storedImages() int {
	e.t.Helper()
	n := 0
	err := filepath.WalkDir(e.photoDir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".jpg") {
			n++
		}
		return err
	})
	if err != nil {
		e.t.Fatal(err)
	}
	return n
}

func TestTripPhotos(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()
	activityID := e.createActivity()
	e.createProfile("jane-doe")
	created, err := e.trips.CreateTripReport(ctx, tripsv1.CreateTripReportRequest_builder{ActivityId: activityID}.Build())
	if err != nil {
		t.Fatal(err)
	}
	reportID := created.GetReport().GetId()
	data := testJPEG(t, 3000, 2000)
	upload := func(b tripsv1.UploadTripPhotoRequest_builder) *tripsv1.TripPhoto {
		t.Helper()
		b.ReportId = reportID
		if b.Data == nil {
			b.Data = data
		}
		resp, err := e.trips.UploadTripPhoto(ctx, b.Build())
		if err != nil {
			t.Fatalf("UploadTripPhoto: %v", err)
		}
		return resp.GetPhoto()
	}

	// The GPS position of a photo is used as is.
	gps := upload(tripsv1.UploadTripPhotoRequest_builder{
		Caption:  " The start ",
		Location: geov1.LatLng_builder{Lat: 0.0181, Lng: 0.0001}.Build(),
	})
	if gps.GetLocationSource() != tripsv1.LocationSource_LOCATION_SOURCE_GPS || gps.GetLocation().GetLat() != 0.0181 ||
		gps.GetTrackDistanceMeters() != 2000 || gps.GetCaption() != "The start" || gps.GetWidth() != 2048 || gps.GetHeight() != 1365 {
		t.Errorf("unexpected GPS photo %v", gps)
	}
	if !strings.HasPrefix(gps.GetUrl(), "https://trail.example/photos/") || !strings.HasSuffix(gps.GetThumbnailUrl(), "/thumb.jpg") {
		t.Errorf("unexpected photo URLs %q, %q", gps.GetUrl(), gps.GetThumbnailUrl())
	}
	// Without one, the photo is placed on the track at the time it was
	// taken: half an hour in, at 5 km.
	timed := upload(tripsv1.UploadTripPhotoRequest_builder{
		TakenTime: timestamppb.New(tripStart.Add(30*time.Minute + 18*time.Second)),
	})
	if timed.GetLocationSource() != tripsv1.LocationSource_LOCATION_SOURCE_TRACK_TIME ||
		!approx(timed.GetLocation().GetLat(), 0.04545, 1e-9) || !approx(timed.GetTrackDistanceMeters(), 5050, 1e-6) {
		t.Errorf("unexpected timed photo %v", timed)
	}
	// Local times are in the trip's timezone, here UTC+2.
	local := upload(tripsv1.UploadTripPhotoRequest_builder{TakenLocalTime: "2026-09-07T10:30:00"})
	if local.GetLocationSource() != tripsv1.LocationSource_LOCATION_SOURCE_TRACK_TIME ||
		!approx(local.GetTrackDistanceMeters(), 5000, 1e-6) || !local.GetTakenTime().AsTime().Equal(tripStart.Add(30*time.Minute)) {
		t.Errorf("unexpected local time photo %v", local)
	}
	// Photos taken away from the track's time aren't placed.
	unplaced := upload(tripsv1.UploadTripPhotoRequest_builder{TakenTime: timestamppb.New(tripStart.Add(24 * time.Hour))})
	if unplaced.HasLocation() || unplaced.GetLocationSource() != tripsv1.LocationSource_LOCATION_SOURCE_UNSPECIFIED || unplaced.HasTrackDistanceMeters() {
		t.Errorf("unexpected unplaced photo %v", unplaced)
	}

	for name, req := range map[string]tripsv1.UploadTripPhotoRequest_builder{
		"empty":      {ReportId: reportID},
		"not image":  {ReportId: reportID, Data: []byte("<svg></svg>")},
		"bad time":   {ReportId: reportID, Data: data, TakenLocalTime: "yesterday"},
		"bad lat":    {ReportId: reportID, Data: data, Location: geov1.LatLng_builder{Lat: 91}.Build()},
		"bad report": {ReportId: "nope", Data: data},
	} {
		_, err := e.trips.UploadTripPhoto(ctx, req.Build())
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("UploadTripPhoto %s: got %v, want invalid argument", name, err)
		}
	}
	_, err = e.otherTrips.UploadTripPhoto(ctx, tripsv1.UploadTripPhotoRequest_builder{ReportId: reportID, Data: data}.Build())
	wantCode(t, err, connect.CodeNotFound)
	if n := e.storedImages(); n != 8 {
		t.Errorf("%d images stored, want 8", n)
	}

	got, err := e.trips.GetTripReport(ctx, tripsv1.GetTripReportRequest_builder{Id: reportID}.Build())
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, p := range got.GetReport().GetPhotos() {
		ids = append(ids, p.GetId())
	}
	if want := []string{gps.GetId(), timed.GetId(), local.GetId(), unplaced.GetId()}; strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("photos in order %v, want %v", ids, want)
	}

	// The author places photos by hand.
	moved, err := e.trips.UpdateTripPhoto(ctx, tripsv1.UpdateTripPhotoRequest_builder{
		Id:       unplaced.GetId(),
		Caption:  new("At the top"),
		Location: geov1.LatLng_builder{Lat: 0.09, Lng: 0}.Build(),
	}.Build())
	if err != nil {
		t.Fatalf("UpdateTripPhoto: %v", err)
	}
	if p := moved.GetPhoto(); p.GetLocationSource() != tripsv1.LocationSource_LOCATION_SOURCE_MANUAL || p.GetTrackDistanceMeters() != 10000 || p.GetCaption() != "At the top" {
		t.Errorf("unexpected moved photo %v", p)
	}
	cleared, err := e.trips.UpdateTripPhoto(ctx, tripsv1.UpdateTripPhotoRequest_builder{Id: timed.GetId(), ClearLocation: true}.Build())
	if err != nil {
		t.Fatalf("UpdateTripPhoto: %v", err)
	}
	if p := cleared.GetPhoto(); p.HasLocation() || p.HasTrackDistanceMeters() || p.GetLocationSource() != tripsv1.LocationSource_LOCATION_SOURCE_UNSPECIFIED {
		t.Errorf("unexpected cleared photo %v", p)
	}
	_, err = e.trips.UpdateTripPhoto(ctx, tripsv1.UpdateTripPhotoRequest_builder{
		Id: timed.GetId(), ClearLocation: true, Location: geov1.LatLng_builder{}.Build(),
	}.Build())
	wantCode(t, err, connect.CodeInvalidArgument)
	_, err = e.otherTrips.UpdateTripPhoto(ctx, tripsv1.UpdateTripPhotoRequest_builder{Id: timed.GetId(), Caption: new("Mine")}.Build())
	wantCode(t, err, connect.CodeNotFound)

	reversed := []string{unplaced.GetId(), local.GetId(), timed.GetId(), gps.GetId()}
	reordered, err := e.trips.ReorderTripPhotos(ctx, tripsv1.ReorderTripPhotosRequest_builder{ReportId: reportID, PhotoIds: reversed}.Build())
	if err != nil {
		t.Fatalf("ReorderTripPhotos: %v", err)
	}
	ids = nil
	for _, p := range reordered.GetPhotos() {
		ids = append(ids, p.GetId())
	}
	if strings.Join(ids, ",") != strings.Join(reversed, ",") {
		t.Errorf("photos in order %v, want %v", ids, reversed)
	}
	for _, photoIDs := range [][]string{reversed[:3], append(reversed, reversed[0]), {reversed[0], reversed[0], reversed[1], reversed[2]}} {
		_, err = e.trips.ReorderTripPhotos(ctx, tripsv1.ReorderTripPhotosRequest_builder{ReportId: reportID, PhotoIds: photoIDs}.Build())
		wantCode(t, err, connect.CodeInvalidArgument)
	}
	_, err = e.otherTrips.ReorderTripPhotos(ctx, tripsv1.ReorderTripPhotosRequest_builder{ReportId: reportID, PhotoIds: reversed}.Build())
	wantCode(t, err, connect.CodeNotFound)

	// Photos of drafts are only served to their author.
	firstURL := unplaced.GetUrl()
	for name, token := range map[string]string{"anonymous": "", "other user": e.otherToken} {
		if resp, _ := e.get(firstURL, token); resp.StatusCode != http.StatusNotFound {
			t.Errorf("draft photo for %s: status %d, want 404", name, resp.StatusCode)
		}
	}
	resp, body := e.get(firstURL, e.token)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "image/jpeg" ||
		!strings.HasPrefix(resp.Header.Get("Cache-Control"), "private") {
		t.Fatalf("draft photo for the author: status %d, headers %v", resp.StatusCode, resp.Header)
	}
	if cfg, err := jpeg.DecodeConfig(bytes.NewReader(body)); err != nil || cfg.Width != 2048 {
		t.Errorf("served photo is %v, %v", cfg, err)
	}
	resp, body = e.get(unplaced.GetThumbnailUrl(), e.token)
	if cfg, err := jpeg.DecodeConfig(bytes.NewReader(body)); resp.StatusCode != http.StatusOK || err != nil || cfg.Width != 480 {
		t.Errorf("served thumbnail: status %d, %v, %v", resp.StatusCode, cfg, err)
	}
	for _, path := range []string{
		strings.Replace(firstURL, "large.jpg", "huge.jpg", 1),
		strings.Replace(firstURL, "large.jpg", "large.png", 1),
		"/photos/00000000-0000-0000-0000-000000000000/large.jpg",
		"/photos/nope/large.jpg",
		"/photos/",
	} {
		if resp, _ := e.get(path, e.token); resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s: status %d, want 404", path, resp.StatusCode)
		}
	}

	// The public page of a draft doesn't exist for others, and isn't
	// indexed for its author.
	reportPath := "/u/jane-doe/ridge-run"
	if resp, _ := e.get(reportPath, ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("draft page: status %d, want 404", resp.StatusCode)
	}
	if resp, body := e.get(reportPath, e.token); resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "noindex") {
		t.Errorf("draft page for the author: status %d\n%s", resp.StatusCode, body)
	}

	if _, err := e.trips.PublishTripReport(ctx, tripsv1.PublishTripReportRequest_builder{Id: reportID}.Build()); err != nil {
		t.Fatal(err)
	}
	resp, _ = e.get(firstURL, "")
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Errorf("published photo: status %d, headers %v", resp.StatusCode, resp.Header)
	}
	if resp, _ := e.get(firstURL, e.otherToken); resp.StatusCode != http.StatusOK {
		t.Errorf("published photo for another user: status %d", resp.StatusCode)
	}

	// Public pages have metadata for search engines and link previews.
	resp, body = e.get(reportPath, "")
	page := string(body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("report page: status %d", resp.StatusCode)
	}
	for _, want := range []string{
		"<title>Ridge run · Jane Doe</title>",
		`<meta name="description" content="Felt good" />`,
		`<meta property="og:image" content="` + firstURL + `" />`,
		`<link rel="canonical" href="https://trail.example/u/jane-doe/ridge-run" />`,
		`<meta property="og:type" content="article" />`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("report page does not contain %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, "noindex") {
		t.Error("published report page has noindex")
	}
	resp, body = e.get("/u/jane-doe", "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "<title>Trip reports by Jane Doe</title>") ||
		!strings.Contains(string(body), `<meta property="og:image" content="`+firstURL+`" />`) {
		t.Errorf("profile page: status %d\n%s", resp.StatusCode, body)
	}
	for _, path := range []string{"/u/nobody", "/u/jane-doe/nothing", "/u/jane-doe/ridge-run/extra", "/u/"} {
		if resp, _ := e.get(path, ""); resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s: status %d, want 404", path, resp.StatusCode)
		}
	}

	// Deleting photos and reports deletes their images.
	if _, err := e.trips.DeleteTripPhoto(ctx, tripsv1.DeleteTripPhotoRequest_builder{Id: gps.GetId()}.Build()); err != nil {
		t.Fatalf("DeleteTripPhoto: %v", err)
	}
	if n := e.storedImages(); n != 6 {
		t.Errorf("%d images stored after deleting a photo, want 6", n)
	}
	_, err = e.otherTrips.DeleteTripPhoto(ctx, tripsv1.DeleteTripPhotoRequest_builder{Id: timed.GetId()}.Build())
	wantCode(t, err, connect.CodeNotFound)
	if _, err := e.trips.DeleteTripReport(ctx, tripsv1.DeleteTripReportRequest_builder{Id: reportID}.Build()); err != nil {
		t.Fatalf("DeleteTripReport: %v", err)
	}
	if n := e.storedImages(); n != 0 {
		t.Errorf("%d images stored after deleting the report, want 0", n)
	}
	if resp, _ := e.get(firstURL, e.token); resp.StatusCode != http.StatusNotFound {
		t.Errorf("photo of a deleted report: status %d, want 404", resp.StatusCode)
	}
}
