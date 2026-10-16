package fit

import (
	"errors"
	"io"
	"math"
	"time"
)

// Global message numbers of workouts.
const (
	workoutMessage     = 26
	workoutStepMessage = 27
)

// SportRunning is the running sport.
const SportRunning = 1

// Intensity is how hard a workout step is, which devices show and record.
type Intensity uint8

// Intensities of workout steps.
const (
	IntensityActive   Intensity = 0
	IntensityRest     Intensity = 1
	IntensityWarmUp   Intensity = 2
	IntensityCoolDown Intensity = 3
)

const (
	// workoutNameSize is the size of workout names, including the
	// terminating zero.
	workoutNameSize = 32
	// stepNameSize is the size of step names, which devices show in a large
	// font, including the terminating zero.
	stepNameSize = 16
	// notesSize is the size of a step's notes, including the terminating
	// zero.
	notesSize = 64
	// MaxWorkoutSteps is the most steps a workout can have. Many Garmin
	// watches accept no more.
	MaxWorkoutSteps = 50
)

// Step duration types.
const (
	durationTime = 0
	durationOpen = 5
)

// Step target types.
const (
	targetHeartRate = 1
	targetOpen      = 2
)

// heartRateOffset is added to custom heart rate targets in beats per minute,
// as values up to 100 are percentages of the maximum heart rate.
const heartRateOffset = 100

// Workout is a structured workout, which devices guide step by step.
type Workout struct {
	Name  string
	Sport uint8
	Steps []WorkoutStep
}

// WorkoutStep is a step of a workout.
type WorkoutStep struct {
	// The name devices show for the step, cut to 15 bytes.
	Name string
	// Notes on the step, which some devices show.
	Notes string
	// How long the step lasts, or 0 for until the lap button is pressed.
	Duration time.Duration
	// The heart rate range to stay in, in bpm, which devices alert on
	// leaving. Both 0 for no target.
	MinHeartRateBPM, MaxHeartRateBPM int
	Intensity                        Intensity
}

// WriteWorkout encodes a workout as a FIT workout file, created at a time.
func WriteWorkout(w io.Writer, workout *Workout, created time.Time) error {
	if len(workout.Steps) == 0 {
		return errors.New("a workout needs at least one step")
	}
	if len(workout.Steps) > MaxWorkoutSteps {
		return errors.New("a workout can have at most 50 steps")
	}
	var f Writer
	f.Define(0, FileIDMessage,
		EnumField(0),   // Type.
		Uint16Field(1), // Manufacturer.
		Uint16Field(2), // Product.
		Uint32Field(4), // Time created.
	)
	f.Data(0, uint8(FileWorkout), uint16(DevelopmentManufacturer), uint16(0), Time(created))
	f.Define(1, workoutMessage,
		EnumField(4),                    // Sport.
		Uint16Field(6),                  // Number of valid steps.
		StringField(8, workoutNameSize), // Name.
	)
	f.Data(1, workout.Sport, uint16(len(workout.Steps)), StringValue(workout.Name, workoutNameSize))
	f.Define(2, workoutStepMessage,
		Uint16Field(254),             // Message index.
		StringField(0, stepNameSize), // Name.
		EnumField(1),                 // Duration type.
		Uint32Field(2),               // Duration value.
		EnumField(3),                 // Target type.
		Uint32Field(4),               // Target value: 0 for a custom range.
		Uint32Field(5),               // Custom target low.
		Uint32Field(6),               // Custom target high.
		EnumField(7),                 // Intensity.
		StringField(8, notesSize),    // Notes.
	)
	for i, step := range workout.Steps {
		durationType, durationValue := uint8(durationOpen), uint32(0)
		if step.Duration > 0 {
			durationType = durationTime
			durationValue = uint32(math.Round(float64(step.Duration / time.Millisecond)))
		}
		targetType, low, high := uint8(targetOpen), uint32(0), uint32(0)
		if step.MaxHeartRateBPM > 0 {
			targetType = targetHeartRate
			low = uint32(step.MinHeartRateBPM + heartRateOffset)
			high = uint32(step.MaxHeartRateBPM + heartRateOffset)
		}
		f.Data(2, uint16(i), StringValue(step.Name, stepNameSize), durationType, durationValue,
			targetType, uint32(0), low, high, uint8(step.Intensity), StringValue(step.Notes, notesSize))
	}
	return f.WriteFile(w)
}
