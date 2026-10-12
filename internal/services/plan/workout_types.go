//lint:file-ignore SA1019 Older clients may still send the deprecated workout types.

package plan

import (
	plansv1 "github.com/catzkorn/trail-tools/internal/gen/plans/v1"
	"github.com/catzkorn/trail-tools/internal/plans"
)

var workoutTypes = map[plansv1.WorkoutType]plans.WorkoutType{
	plansv1.WorkoutType_WORKOUT_TYPE_BASE:           plans.WorkoutTypeBase,
	plansv1.WorkoutType_WORKOUT_TYPE_INTENSITY:      plans.WorkoutTypeIntensity,
	plansv1.WorkoutType_WORKOUT_TYPE_CROSS_TRAINING: plans.WorkoutTypeCrossTraining,
	plansv1.WorkoutType_WORKOUT_TYPE_STRENGTH:       plans.WorkoutTypeStrength,
	plansv1.WorkoutType_WORKOUT_TYPE_RACE:           plans.WorkoutTypeRace,
	plansv1.WorkoutType_WORKOUT_TYPE_REST:           plans.WorkoutTypeRest,
}

// deprecatedWorkoutTypes are the types older clients may still send, by the
// type that replaced them.
var deprecatedWorkoutTypes = map[plansv1.WorkoutType]plans.WorkoutType{
	plansv1.WorkoutType_WORKOUT_TYPE_EASY:         plans.WorkoutTypeBase,
	plansv1.WorkoutType_WORKOUT_TYPE_LONG_RUN:     plans.WorkoutTypeBase,
	plansv1.WorkoutType_WORKOUT_TYPE_RECOVERY:     plans.WorkoutTypeBase,
	plansv1.WorkoutType_WORKOUT_TYPE_VERT:         plans.WorkoutTypeBase,
	plansv1.WorkoutType_WORKOUT_TYPE_TEMPO:        plans.WorkoutTypeIntensity,
	plansv1.WorkoutType_WORKOUT_TYPE_INTERVALS:    plans.WorkoutTypeIntensity,
	plansv1.WorkoutType_WORKOUT_TYPE_HILL_REPEATS: plans.WorkoutTypeIntensity,
	plansv1.WorkoutType_WORKOUT_TYPE_OTHER:        plans.WorkoutTypeCrossTraining,
}

// workoutTypeFromProto returns the stored type of an API workout type,
// mapping deprecated types to the ones that replaced them.
func workoutTypeFromProto(t plansv1.WorkoutType) (plans.WorkoutType, bool) {
	if wt, ok := workoutTypes[t]; ok {
		return wt, true
	}
	wt, ok := deprecatedWorkoutTypes[t]
	return wt, ok
}

func workoutTypeToProto(t plans.WorkoutType) plansv1.WorkoutType {
	for k, v := range workoutTypes {
		if v == t {
			return k
		}
	}
	return plansv1.WorkoutType_WORKOUT_TYPE_UNSPECIFIED
}
