package services_test

import (
	"context"
	"testing"

	"connectrpc.com/connect/v2"
	geov1 "github.com/catzkorn/trail-tools/internal/gen/geo/v1"
	plansv1 "github.com/catzkorn/trail-tools/internal/gen/plans/v1"
	racesv1 "github.com/catzkorn/trail-tools/internal/gen/races/v1"
	routesv1 "github.com/catzkorn/trail-tools/internal/gen/routes/v1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestTrainingPlans(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()

	race, err := e.races.CreateRace(ctx, racesv1.CreateRaceRequest_builder{
		Race: racesv1.Race_builder{Name: "Goal race", Date: "2027-03-28"}.Build(),
	}.Build())
	if err != nil {
		t.Fatal(err)
	}
	raceID := race.GetRace().GetId()

	// Plans are at most a year long, and goal races must exist.
	_, err = e.plans.CreateTrainingPlan(ctx, plansv1.CreateTrainingPlanRequest_builder{
		Plan: plansv1.TrainingPlan_builder{Name: "Too long", StartDate: "2026-01-01", EndDate: "2027-06-01"}.Build(),
	}.Build())
	wantCode(t, err, connect.CodeInvalidArgument)
	_, err = e.plans.CreateTrainingPlan(ctx, plansv1.CreateTrainingPlanRequest_builder{
		Plan: plansv1.TrainingPlan_builder{
			Name: "Missing race", StartDate: "2026-11-01", EndDate: "2027-03-28",
			GoalRaceIds: []string{"00000000-0000-0000-0000-000000000000"},
		}.Build(),
	}.Build())
	wantCode(t, err, connect.CodeInvalidArgument)

	created, err := e.plans.CreateTrainingPlan(ctx, plansv1.CreateTrainingPlanRequest_builder{
		Plan: plansv1.TrainingPlan_builder{
			Name:        "Spring ultra",
			Description: "20 weeks to the goal race",
			StartDate:   "2026-11-02",
			EndDate:     "2027-03-28",
			GoalRaceIds: []string{raceID},
		}.Build(),
	}.Build())
	if err != nil {
		t.Fatalf("CreateTrainingPlan: %v", err)
	}
	planID := created.GetPlan().GetId()
	// The goal race is on the calendar, as the plan trains for it.
	if goal, err := e.races.GetRace(ctx, racesv1.GetRaceRequest_builder{Id: raceID}.Build()); err != nil || !goal.GetRace().GetOnCalendar() {
		t.Errorf("the goal race isn't on the calendar (%v)", err)
	}

	// Periods are normalised to the start of the month or week.
	periods, err := e.plans.SetPlanPeriods(ctx, plansv1.SetPlanPeriodsRequest_builder{
		PlanId: planID,
		Periods: []*plansv1.PlanPeriod{
			plansv1.PlanPeriod_builder{
				Granularity: plansv1.PeriodGranularity_PERIOD_GRANULARITY_MONTH,
				StartDate:   "2026-11-15",
				Title:       "Base",
			}.Build(),
			plansv1.PlanPeriod_builder{
				Granularity:          plansv1.PeriodGranularity_PERIOD_GRANULARITY_WEEK,
				StartDate:            "2026-11-05",
				Title:                "Base 1",
				TargetDistanceMeters: 50000,
			}.Build(),
		},
	}.Build())
	if err != nil {
		t.Fatalf("SetPlanPeriods: %v", err)
	}
	if got := periods.GetPeriods(); got[0].GetStartDate() != "2026-11-01" || got[1].GetStartDate() != "2026-11-02" {
		t.Errorf("unexpected period dates %v", got)
	}
	// Setting a period again replaces it.
	if _, err := e.plans.SetPlanPeriods(ctx, plansv1.SetPlanPeriodsRequest_builder{
		PlanId: planID,
		Periods: []*plansv1.PlanPeriod{plansv1.PlanPeriod_builder{
			Granularity:          plansv1.PeriodGranularity_PERIOD_GRANULARITY_WEEK,
			StartDate:            "2026-11-02",
			Title:                "Base 1 (revised)",
			TargetDistanceMeters: 45000,
		}.Build()},
	}.Build()); err != nil {
		t.Fatalf("SetPlanPeriods: %v", err)
	}
	plan, err := e.plans.GetTrainingPlan(ctx, plansv1.GetTrainingPlanRequest_builder{Id: planID}.Build())
	if err != nil {
		t.Fatalf("GetTrainingPlan: %v", err)
	}
	if got := plan.GetPeriods(); len(got) != 2 || got[1].GetTitle() != "Base 1 (revised)" || got[1].GetTargetDistanceMeters() != 45000 {
		t.Errorf("unexpected periods %v", got)
	}
	_, err = e.plans.SetPlanPeriods(ctx, plansv1.SetPlanPeriodsRequest_builder{
		PlanId: planID,
		Periods: []*plansv1.PlanPeriod{plansv1.PlanPeriod_builder{
			Granularity: plansv1.PeriodGranularity_PERIOD_GRANULARITY_MONTH,
			StartDate:   "2027-06-01",
		}.Build()},
	}.Build())
	wantCode(t, err, connect.CodeInvalidArgument)

	// Create workouts in bulk, one with a route.
	route, err := e.routes.CreateRoute(ctx, routesv1.CreateRouteRequest_builder{
		Name:      "Long run loop",
		Waypoints: routesv1.Waypoints_builder{Waypoints: latLngs(0, 0, 0.1, 0)}.Build(),
	}.Build())
	if err != nil {
		t.Fatal(err)
	}
	workouts, err := e.plans.CreateWorkouts(ctx, plansv1.CreateWorkoutsRequest_builder{
		PlanId: planID,
		Workouts: []*plansv1.Workout{
			plansv1.Workout_builder{Date: "2026-11-03", Type: plansv1.WorkoutType_WORKOUT_TYPE_BASE, Title: "Easy 8k", TargetDistanceMeters: 8000}.Build(),
			plansv1.Workout_builder{
				// Older clients may send deprecated types.
				//lint:ignore SA1019 Testing a deprecated type.
				Date: "2026-11-08", Type: plansv1.WorkoutType_WORKOUT_TYPE_LONG_RUN, Title: "Long run",
				TargetElevationGainMeters: 1000, TargetTerrain: geov1.Terrain_TERRAIN_TRAIL, RouteId: route.GetRoute().GetId(),
			}.Build(),
			plansv1.Workout_builder{Date: "2026-11-10", Type: plansv1.WorkoutType_WORKOUT_TYPE_REST}.Build(),
		},
	}.Build())
	if err != nil {
		t.Fatalf("CreateWorkouts: %v", err)
	}
	longRun := workouts.GetWorkouts()[1]
	if longRun.GetPlanId() != planID || longRun.GetRouteId() != route.GetRoute().GetId() || longRun.GetTargetTerrain() != geov1.Terrain_TERRAIN_TRAIL {
		t.Errorf("unexpected workout %v", longRun)
	}
	if longRun.GetType() != plansv1.WorkoutType_WORKOUT_TYPE_BASE {
		t.Errorf("a long run has type %v, want base", longRun.GetType())
	}
	// Workouts must be inside the plan and have a type, and the batch is
	// atomic.
	_, err = e.plans.CreateWorkouts(ctx, plansv1.CreateWorkoutsRequest_builder{
		PlanId: planID,
		Workouts: []*plansv1.Workout{
			plansv1.Workout_builder{Date: "2026-11-04", Type: plansv1.WorkoutType_WORKOUT_TYPE_BASE}.Build(),
			plansv1.Workout_builder{Date: "2026-10-01", Type: plansv1.WorkoutType_WORKOUT_TYPE_BASE}.Build(),
		},
	}.Build())
	wantCode(t, err, connect.CodeInvalidArgument)

	// List the first week.
	list, err := e.plans.ListWorkouts(ctx, plansv1.ListWorkoutsRequest_builder{
		PlanId: planID, StartDate: "2026-11-02", EndDate: "2026-11-08",
	}.Build())
	if err != nil {
		t.Fatalf("ListWorkouts: %v", err)
	}
	if len(list.GetWorkouts()) != 2 || list.GetWorkouts()[0].GetTitle() != "Easy 8k" {
		t.Errorf("unexpected workouts %v", list.GetWorkouts())
	}
	// Without a plan, the workouts of all the user's plans are listed, and
	// no one else's.
	all, err := e.plans.ListWorkouts(ctx, plansv1.ListWorkoutsRequest_builder{
		StartDate: "2026-11-02", EndDate: "2026-11-08",
	}.Build())
	if err != nil || len(all.GetWorkouts()) != 2 {
		t.Errorf("got %d workouts of all plans (%v), want 2", len(all.GetWorkouts()), err)
	}
	others, err := e.otherTraining.ListWorkouts(ctx, plansv1.ListWorkoutsRequest_builder{}.Build())
	if err != nil || len(others.GetWorkouts()) != 0 {
		t.Errorf("another user listed %d workouts (%v), want none", len(others.GetWorkouts()), err)
	}

	// Move the long run a day and unassign its route.
	moved, err := e.plans.UpdateWorkout(ctx, plansv1.UpdateWorkoutRequest_builder{
		Workout:    plansv1.Workout_builder{Id: longRun.GetId(), Date: "2026-11-07", Title: "ignored"}.Build(),
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"date", "route_id"}},
	}.Build())
	if err != nil {
		t.Fatalf("UpdateWorkout: %v", err)
	}
	if w := moved.GetWorkout(); w.GetDate() != "2026-11-07" || w.GetRouteId() != "" || w.GetTitle() != "Long run" {
		t.Errorf("unexpected moved workout %v", w)
	}

	got, err := e.plans.GetWorkout(ctx, plansv1.GetWorkoutRequest_builder{Id: longRun.GetId()}.Build())
	if err != nil {
		t.Fatalf("GetWorkout: %v", err)
	}
	if got.GetWorkout().GetDate() != "2026-11-07" {
		t.Errorf("unexpected workout %v", got.GetWorkout())
	}

	// Update the plan's description only.
	updated, err := e.plans.UpdateTrainingPlan(ctx, plansv1.UpdateTrainingPlanRequest_builder{
		Plan:       plansv1.TrainingPlan_builder{Id: planID, Description: "Revised"}.Build(),
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
	}.Build())
	if err != nil {
		t.Fatalf("UpdateTrainingPlan: %v", err)
	}
	if p := updated.GetPlan(); p.GetDescription() != "Revised" || p.GetName() != "Spring ultra" || len(p.GetGoalRaceIds()) != 1 {
		t.Errorf("unexpected plan %v", p)
	}

	// Deleting the goal race removes it from the plan.
	if _, err := e.races.DeleteRace(ctx, racesv1.DeleteRaceRequest_builder{Id: raceID}.Build()); err != nil {
		t.Fatal(err)
	}
	plan, err = e.plans.GetTrainingPlan(ctx, plansv1.GetTrainingPlanRequest_builder{Id: planID}.Build())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.GetPlan().GetGoalRaceIds()) != 0 {
		t.Errorf("goal race was not removed: %v", plan.GetPlan())
	}

	if _, err := e.plans.DeleteWorkouts(ctx, plansv1.DeleteWorkoutsRequest_builder{
		PlanId: planID, Ids: []string{workouts.GetWorkouts()[2].GetId()},
	}.Build()); err != nil {
		t.Fatalf("DeleteWorkouts: %v", err)
	}
	if _, err := e.plans.DeleteTrainingPlan(ctx, plansv1.DeleteTrainingPlanRequest_builder{Id: planID}.Build()); err != nil {
		t.Fatalf("DeleteTrainingPlan: %v", err)
	}
	_, err = e.plans.ListWorkouts(ctx, plansv1.ListWorkoutsRequest_builder{PlanId: planID}.Build())
	wantCode(t, err, connect.CodeNotFound)
}
