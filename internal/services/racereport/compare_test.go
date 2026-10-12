package racereport

import (
	"testing"
	"time"

	activitiesv1 "github.com/catzkorn/trail-tools/internal/gen/activities/v1"
	racesv1 "github.com/catzkorn/trail-tools/internal/gen/races/v1"
	"github.com/catzkorn/trail-tools/internal/races"
	"github.com/catzkorn/trail-tools/internal/services/raceplan"
)

// testRecording is a recording 1 km longer than a 20 km race, as GPS
// distances often are, at 6 minutes per kilometer.
func testRecording() *recording {
	streams := &activitiesv1.ActivityStreams{}
	for km := 0; km <= 21; km++ {
		streams.SetTimeSeconds(append(streams.GetTimeSeconds(), int32(km*360)))
		streams.SetDistanceMeters(append(streams.GetDistanceMeters(), float64(km*1000)))
	}
	return newRecording(streams)
}

func TestElapsedAt(t *testing.T) {
	t.Parallel()
	rec := testRecording()
	for _, tt := range []struct {
		meters float64
		want   int32
		ok     bool
	}{
		{0, 0, true},
		{500, 180, true},
		{10_000, 3600, true},
		{21_000, 7560, true},
		{21_001, 0, false},
	} {
		got, ok := rec.elapsedAt(tt.meters)
		if got != tt.want || ok != tt.ok {
			t.Errorf("elapsedAt(%v) = %v, %v, want %v, %v", tt.meters, got, ok, tt.want, tt.ok)
		}
	}
	if newRecording(&activitiesv1.ActivityStreams{}) != nil {
		t.Error("a recording without times was created")
	}
}

func TestCompare(t *testing.T) {
	t.Parallel()
	start := time.Date(2027, 6, 26, 5, 0, 0, 0, time.UTC)
	outcome := &raceplan.PlanOutcome{
		Goals:                races.RaceGoals{AFinishSeconds: 7000, BFinishSeconds: 7500, CFinishSeconds: 8000},
		PlannedFinishSeconds: 7200,
		Checkpoints: []raceplan.PlannedCheckpoint{
			{Name: "Halfway", DistanceMeters: 10_000, PlannedArrivalSeconds: 3500, RecordedArrival: start.Add(3700 * time.Second)},
			{Name: "Finish", DistanceMeters: 20_000, PlannedArrivalSeconds: 7200, CutoffSeconds: 9000},
		},
		RaceDayStart: start,
	}

	// The finish time is compared with the goals, and the checkpoints with
	// the recording, scaled to the race distance: halfway is at 10.5 km of
	// the recording.
	got := compare(outcome, 7560, 20_000, testRecording())
	goals := got.GetGoals()
	if !got.GetHasPlan() || len(goals) != 3 || goals[0].GetMet() || goals[0].GetDifferenceSeconds() != 560 || goals[1].GetMet() || !goals[2].GetMet() {
		t.Errorf("unexpected goals %v", goals)
	}
	checkpoints := got.GetCheckpoints()
	if got.GetActualTimesSource() != racesv1.ActualTimesSource_ACTUAL_TIMES_SOURCE_ACTIVITY || len(checkpoints) != 2 {
		t.Fatalf("unexpected comparison %v", got)
	}
	if cp := checkpoints[0]; cp.GetActualArrivalSeconds() != 3780 || cp.GetDifferenceSeconds() != 280 {
		t.Errorf("unexpected halfway %v", cp)
	}
	if cp := checkpoints[1]; cp.GetActualArrivalSeconds() != 7560 || cp.GetDifferenceSeconds() != 360 || cp.GetCutoffSeconds() != 9000 {
		t.Errorf("unexpected finish %v", cp)
	}

	// Without a recording, the race day splits are used, where recorded.
	got = compare(outcome, 0, 20_000, nil)
	checkpoints = got.GetCheckpoints()
	if got.GetActualTimesSource() != racesv1.ActualTimesSource_ACTUAL_TIMES_SOURCE_RACE_DAY ||
		checkpoints[0].GetActualArrivalSeconds() != 3700 || checkpoints[1].HasActualArrivalSeconds() {
		t.Errorf("unexpected comparison from race day %v", got)
	}
	// Without a finish time, no goal is met.
	for _, goal := range got.GetGoals() {
		if goal.GetMet() || goal.GetDifferenceSeconds() != 0 {
			t.Errorf("unexpected goal without a finish time %v", goal)
		}
	}

	// Without a plan, there is nothing to compare.
	if got := compare(nil, 7560, 20_000, testRecording()); got.GetHasPlan() || len(got.GetCheckpoints()) != 0 {
		t.Errorf("unexpected comparison without a plan %v", got)
	}
}
