package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	tripsv1 "github.com/catzkorn/trail-tools/internal/gen/trips/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// createHillReport creates a report from an activity that climbs from
// 500 m to 1,300 m in 5 km and goes back down, and returns its ID and its
// first section.
func (e *env) createHillReport() (string, *tripsv1.TripSection) {
	e.t.Helper()
	altitudes := make([]float64, 101)
	for i := range altitudes {
		altitudes[i] = 1300 - 16*float64(max(50-i, i-50))
	}
	activityID := e.createActivityWithAltitudes(altitudes)
	created, err := e.trips.CreateTripReport(context.Background(), tripsv1.CreateTripReportRequest_builder{ActivityId: activityID}.Build())
	if err != nil {
		e.t.Fatal(err)
	}
	return created.GetReport().GetId(), created.GetReport().GetSections()[0]
}

// uploadPhoto uploads a photo to a report, taken at the time if it's set,
// and returns its ID.
func (e *env) uploadPhoto(reportID string, takenTime *timestamppb.Timestamp) string {
	e.t.Helper()
	resp, err := e.trips.UploadTripPhoto(context.Background(), tripsv1.UploadTripPhotoRequest_builder{
		ReportId:  reportID,
		Data:      testJPEG(e.t, 300, 200),
		TakenTime: takenTime,
	}.Build())
	if err != nil {
		e.t.Fatalf("UploadTripPhoto: %v", err)
	}
	return resp.GetPhoto().GetId()
}

func TestTripSections(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()
	e.createProfile("jane-doe")
	reportID, intro := e.createHillReport()
	// At 2 and 8 km, and one that isn't on the track.
	at8km := e.uploadPhoto(reportID, timestamppb.New(tripStart.Add(48*time.Minute)))
	at2km := e.uploadPhoto(reportID, timestamppb.New(tripStart.Add(12*time.Minute)))
	unplaced := e.uploadPhoto(reportID, nil)

	// An introduction, the climb and the descent.
	set := func(sections ...*tripsv1.TripSection) (*tripsv1.TripReport, error) {
		resp, err := e.trips.SetTripSections(ctx, tripsv1.SetTripSectionsRequest_builder{ReportId: reportID, Sections: sections}.Build())
		return resp.GetReport(), err
	}
	report, err := set(
		tripsv1.TripSection_builder{Id: intro.GetId(), Body: "Felt good"}.Build(),
		tripsv1.TripSection_builder{
			Title: " Up ", Body: "Up we go.", StartDistanceMeters: new(0.0), EndDistanceMeters: new(5000.0),
			PhotoIds: []string{unplaced, at2km},
		}.Build(),
		tripsv1.TripSection_builder{
			Title: "Down", Body: "And down again.", StartDistanceMeters: new(5000.0), EndDistanceMeters: new(10000.0),
			PhotoIds: []string{at8km},
		}.Build(),
	)
	if err != nil {
		t.Fatalf("SetTripSections: %v", err)
	}
	got := report.GetSections()
	if len(got) != 3 || got[0].GetId() != intro.GetId() || got[0].GetPhotoIds() != nil || got[1].GetTitle() != "Up" ||
		got[1].GetBody() != "Up we go." || got[1].GetEndDistanceMeters() != 5000 || got[2].GetBody() != "And down again." || got[1].GetId() == "" {
		t.Fatalf("unexpected sections %v", got)
	}
	if ids := photoIDs(report); ids != unplaced+","+at2km+","+at8km {
		t.Errorf("got photos %s, want them in the order of the story", ids)
	}
	if report.GetExcerpt() != "Felt good" {
		t.Errorf("got excerpt %q, want the introduction", report.GetExcerpt())
	}

	// New photos go in the section about where they were taken, or else the
	// last one.
	at3km := e.uploadPhoto(reportID, timestamppb.New(tripStart.Add(18*time.Minute)))
	another := e.uploadPhoto(reportID, nil)
	resp, err := e.trips.GetTripReport(ctx, tripsv1.GetTripReportRequest_builder{Id: reportID}.Build())
	if err != nil {
		t.Fatal(err)
	}
	got = resp.GetReport().GetSections()
	if ids := strings.Join(got[1].GetPhotoIds(), ","); ids != unplaced+","+at2km+","+at3km {
		t.Errorf("got photos %s in the climb, want the new one at 3 km last", ids)
	}
	if ids := strings.Join(got[2].GetPhotoIds(), ","); ids != at8km+","+another {
		t.Errorf("got photos %s in the descent, want the new one without a place last", ids)
	}

	// Removing a section moves its photos to the first section, unless
	// they're listed elsewhere.
	got[0].SetPhotoIds([]string{at3km})
	report, err = set(got[0], got[2])
	if err != nil {
		t.Fatalf("SetTripSections: %v", err)
	}
	if s := report.GetSections(); len(s) != 2 || strings.Join(s[0].GetPhotoIds(), ",") != at3km+","+unplaced+","+at2km {
		t.Errorf("got sections %v, want the climb's photos in the introduction", s)
	}

	// Lists show an excerpt, without the sections.
	list, err := e.trips.ListTripReports(ctx, tripsv1.ListTripReportsRequest_builder{}.Build())
	if err != nil {
		t.Fatal(err)
	}
	if r := list.GetReports()[0]; r.GetExcerpt() != "Felt good" || len(r.GetSections()) != 0 {
		t.Errorf("unexpected listed report %v", r)
	}
	// Public pages include the sections.
	public, err := e.trips.GetPublicTripReport(ctx, tripsv1.GetPublicTripReportRequest_builder{Handle: "jane-doe", Slug: "ridge-run"}.Build())
	if err != nil {
		t.Fatal(err)
	}
	if s := public.GetReport().GetSections(); len(s) != 2 || s[1].GetBody() != "And down again." {
		t.Errorf("got public sections %v", s)
	}

	standalone, err := e.trips.CreateTripReport(ctx, tripsv1.CreateTripReportRequest_builder{Title: "No track"}.Build())
	if err != nil {
		t.Fatal(err)
	}
	otherSection := standalone.GetReport().GetSections()[0].GetId()
	for name, sections := range map[string][]*tripsv1.TripSection{
		"none":             nil,
		"start only":       {tripsv1.TripSection_builder{StartDistanceMeters: new(0.0)}.Build()},
		"end before start": {tripsv1.TripSection_builder{StartDistanceMeters: new(500.0), EndDistanceMeters: new(500.0)}.Build()},
		"negative":         {tripsv1.TripSection_builder{StartDistanceMeters: new(-1.0), EndDistanceMeters: new(500.0)}.Build()},
		"long title":       {tripsv1.TripSection_builder{Title: strings.Repeat("a", 201)}.Build()},
		"other report":     {tripsv1.TripSection_builder{Id: otherSection}.Build()},
		"twice":            {got[0], got[0]},
		"photo twice":      {tripsv1.TripSection_builder{PhotoIds: []string{at2km, at2km}}.Build()},
		"bad photo":        {tripsv1.TripSection_builder{PhotoIds: []string{otherSection}}.Build()},
	} {
		if _, err := set(sections...); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("SetTripSections with %s: got %v, want invalid argument", name, err)
		}
	}
	_, err = e.otherTrips.SetTripSections(ctx, tripsv1.SetTripSectionsRequest_builder{
		ReportId: reportID,
		Sections: []*tripsv1.TripSection{tripsv1.TripSection_builder{Body: "Mine"}.Build()},
	}.Build())
	wantCode(t, err, connect.CodeNotFound)
}

// photoIDs returns the IDs of a report's photos, separated by commas.
func photoIDs(report *tripsv1.TripReport) string {
	ids := make([]string, len(report.GetPhotos()))
	for i, p := range report.GetPhotos() {
		ids[i] = p.GetId()
	}
	return strings.Join(ids, ",")
}
