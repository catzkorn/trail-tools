package services_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	racesv1 "github.com/catzkorn/trail-tools/internal/gen/races/v1"
	routesv1 "github.com/catzkorn/trail-tools/internal/gen/routes/v1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestRaces(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()

	// A 22km course climbing 200m.
	course, err := e.routes.CreateRoute(ctx, connect.NewRequest(routesv1.CreateRouteRequest_builder{
		Name:      "Course",
		Waypoints: routesv1.Waypoints_builder{Waypoints: latLngs(0, 0, 0.1, 0, 0.2, 0)}.Build(),
	}.Build()))
	if err != nil {
		t.Fatal(err)
	}
	created, err := e.races.CreateRace(ctx, connect.NewRequest(racesv1.CreateRaceRequest_builder{
		Race: racesv1.Race_builder{
			Name:           "Mountain Ultra",
			Date:           "2027-06-12",
			StartTime:      "5:00",
			DistanceMeters: 22239,
			CutoffSeconds:  6 * 3600,
			Priority:       racesv1.RacePriority_RACE_PRIORITY_A,
			CourseRouteId:  course.Msg.GetRoute().GetId(),
		}.Build(),
	}.Build()))
	if err != nil {
		t.Fatalf("CreateRace: %v", err)
	}
	raceID := created.Msg.GetRace().GetId()
	if created.Msg.GetRace().GetStartTime() != "05:00" {
		t.Errorf("start time should be normalised, got %q", created.Msg.GetRace().GetStartTime())
	}

	// Validation errors.
	_, err = e.races.CreateRace(ctx, connect.NewRequest(racesv1.CreateRaceRequest_builder{
		Race: racesv1.Race_builder{Name: "Bad", Date: "12/06/2027"}.Build(),
	}.Build()))
	wantCode(t, err, connect.CodeInvalidArgument)

	// Races are private.
	_, err = e.otherRaces.GetRace(ctx, connect.NewRequest(racesv1.GetRaceRequest_builder{Id: raceID}.Build()))
	wantCode(t, err, connect.CodeNotFound)

	// Update only the notes.
	updated, err := e.races.UpdateRace(ctx, connect.NewRequest(racesv1.UpdateRaceRequest_builder{
		Race:       racesv1.Race_builder{Id: raceID, Notes: "Bring poles", Name: "ignored"}.Build(),
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"notes"}},
	}.Build()))
	if err != nil {
		t.Fatalf("UpdateRace: %v", err)
	}
	if updated.Msg.GetRace().GetNotes() != "Bring poles" || updated.Msg.GetRace().GetName() != "Mountain Ultra" {
		t.Errorf("unexpected race %v", updated.Msg.GetRace())
	}

	// Deleting a race removes it.
	if _, err := e.races.DeleteRace(ctx, connect.NewRequest(racesv1.DeleteRaceRequest_builder{Id: raceID}.Build())); err != nil {
		t.Fatalf("DeleteRace: %v", err)
	}
	_, err = e.races.GetRace(ctx, connect.NewRequest(racesv1.GetRaceRequest_builder{Id: raceID}.Build()))
	wantCode(t, err, connect.CodeNotFound)
}
