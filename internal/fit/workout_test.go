package fit_test

import (
	"bytes"
	"slices"
	"testing"
	"time"

	"github.com/johanbrandhorst/trail-tools/internal/fit"
	"github.com/johanbrandhorst/trail-tools/internal/fit/fittest"
)

func TestWriteWorkout(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	err := fit.WriteWorkout(&buf, &fit.Workout{
		Name:  "Lactate test",
		Sport: fit.SportRunning,
		Steps: []fit.WorkoutStep{
			{Name: "Warm up", Duration: 15 * time.Minute, Intensity: fit.IntensityWarmUp},
			{Name: "100 bpm", Notes: "Hold 95-105 bpm", Duration: 4 * time.Minute, MinHeartRateBPM: 95, MaxHeartRateBPM: 105},
			{Name: "Sample", Intensity: fit.IntensityRest},
		},
	}, created)
	if err != nil {
		t.Fatalf("WriteWorkout: %v", err)
	}
	messages := fittest.Decode(t, buf.Bytes())
	if len(messages) != 5 {
		t.Fatalf("got %d messages, want a file ID, a workout and 3 steps", len(messages))
	}
	if fileID := messages[0]; fileID.Uint(t, 0) != fit.FileWorkout || fileID.Uint(t, 4) != uint64(fit.Time(created)) {
		t.Errorf("unexpected file ID %v", fileID.Fields)
	}
	if workout := messages[1]; workout.String(8) != "Lactate test" || workout.Uint(t, 4) != fit.SportRunning || workout.Uint(t, 6) != 3 {
		t.Errorf("unexpected workout %v", workout.Fields)
	}
	for i, want := range []struct {
		name                     string
		durationType, durationMS uint64
		targetType, low, high    uint64
		intensity                uint64
	}{
		// Warm up for 15 minutes with no target.
		{"Warm up", 0, 15 * 60 * 1000, 2, 0, 0, 2},
		// Hold 95 to 105 bpm for 4 minutes, as heart rates offset by 100.
		{"100 bpm", 0, 4 * 60 * 1000, 1, 195, 205, 0},
		// Sample until the lap button is pressed.
		{"Sample", 5, 0, 2, 0, 0, 1},
	} {
		step := messages[2+i]
		got := []uint64{step.Uint(t, 1), step.Uint(t, 2), step.Uint(t, 3), step.Uint(t, 5), step.Uint(t, 6), step.Uint(t, 7)}
		wantValues := []uint64{want.durationType, want.durationMS, want.targetType, want.low, want.high, want.intensity}
		if step.String(0) != want.name || step.Uint(t, 254) != uint64(i) || !slices.Equal(got, wantValues) {
			t.Errorf("step %d: got %q %v, want %q %v", i, step.String(0), got, want.name, wantValues)
		}
	}
	if notes := messages[3].String(8); notes != "Hold 95-105 bpm" {
		t.Errorf("got notes %q", notes)
	}

	if err := fit.WriteWorkout(&buf, &fit.Workout{Name: "Too long", Steps: make([]fit.WorkoutStep, fit.MaxWorkoutSteps+1)}, created); err == nil {
		t.Error("wrote a workout with too many steps")
	}
}
