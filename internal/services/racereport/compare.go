package racereport

import (
	"math"
	"sort"

	activitiesv1 "github.com/catzkorn/trail-tools/internal/gen/activities/v1"
	racesv1 "github.com/catzkorn/trail-tools/internal/gen/races/v1"
	"github.com/catzkorn/trail-tools/internal/services/raceplan"
)

// recording is an activity's elapsed time by distance.
type recording struct {
	times     []int32
	distances []float64
}

// newRecording returns the recording of an activity's streams, or nil if
// they have no times or distances.
func newRecording(streams *activitiesv1.ActivityStreams) *recording {
	times, distances := streams.GetTimeSeconds(), streams.GetDistanceMeters()
	if len(times) < 2 || len(times) != len(distances) {
		return nil
	}
	return &recording{times: times, distances: distances}
}

// totalMeters returns the recorded distance.
func (r *recording) totalMeters() float64 {
	return r.distances[len(r.distances)-1]
}

// elapsedAt returns the elapsed time in seconds when the recording first
// reached a distance, interpolated between samples. ok is false beyond the
// recorded distance.
func (r *recording) elapsedAt(meters float64) (seconds int32, ok bool) {
	if meters > r.totalMeters() {
		return 0, false
	}
	i := sort.SearchFloat64s(r.distances, meters)
	if i == 0 {
		return r.times[0], true
	}
	before, after := r.distances[i-1], r.distances[i]
	f := 0.0
	if after > before {
		f = (meters - before) / (after - before)
	}
	elapsed := float64(r.times[i-1]) + f*float64(r.times[i]-r.times[i-1])
	return int32(math.Round(elapsed)), true
}

// compare compares a race plan with what happened: the finish time against
// the goals, and the arrival times at the checkpoints, from the recording if
// any, scaled to the race distance as GPS distances differ from course
// distances, or else from the race day splits.
func compare(outcome *raceplan.PlanOutcome, finishSeconds int32, raceMeters float64, rec *recording) *racesv1.RaceComparison {
	b := racesv1.RaceComparison_builder{}
	if outcome == nil {
		return b.Build()
	}
	b.HasPlan = true
	b.PlannedFinishSeconds = outcome.PlannedFinishSeconds
	for _, goal := range []struct {
		label  string
		target int32
	}{
		{"A", outcome.Goals.AFinishSeconds},
		{"B", outcome.Goals.BFinishSeconds},
		{"C", outcome.Goals.CFinishSeconds},
	} {
		if goal.target <= 0 {
			continue
		}
		result := racesv1.GoalResult_builder{Goal: goal.label, TargetSeconds: goal.target}
		if finishSeconds > 0 {
			result.Met = finishSeconds <= goal.target
			result.DifferenceSeconds = finishSeconds - goal.target
		}
		b.Goals = append(b.Goals, result.Build())
	}

	if raceMeters <= 0 && len(outcome.Checkpoints) > 0 {
		raceMeters = outcome.Checkpoints[len(outcome.Checkpoints)-1].DistanceMeters
	}
	scale := 1.0
	if rec != nil && raceMeters > 0 {
		scale = rec.totalMeters() / raceMeters
	}
	hasRaceDay := !outcome.RaceDayStart.IsZero()
	for _, cp := range outcome.Checkpoints {
		result := racesv1.CheckpointResult_builder{
			Name:                  cp.Name,
			DistanceMeters:        cp.DistanceMeters,
			PlannedArrivalSeconds: cp.PlannedArrivalSeconds,
			CutoffSeconds:         cp.CutoffSeconds,
		}
		var actual int32
		found := false
		switch {
		case rec != nil:
			actual, found = rec.elapsedAt(cp.DistanceMeters * scale)
			b.ActualTimesSource = racesv1.ActualTimesSource_ACTUAL_TIMES_SOURCE_ACTIVITY
		case hasRaceDay && !cp.RecordedArrival.IsZero():
			actual = int32(cp.RecordedArrival.Sub(outcome.RaceDayStart).Seconds())
			found = true
			b.ActualTimesSource = racesv1.ActualTimesSource_ACTUAL_TIMES_SOURCE_RACE_DAY
		}
		if found {
			difference := actual - cp.PlannedArrivalSeconds
			result.ActualArrivalSeconds = &actual
			result.DifferenceSeconds = &difference
		}
		b.Checkpoints = append(b.Checkpoints, result.Build())
	}
	return b.Build()
}
