package services_test

import (
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect/v2"
	"github.com/johanbrandhorst/trail-tools/internal/fit/fittest"
	lactatev1 "github.com/johanbrandhorst/trail-tools/internal/gen/lactate/v1"
	lactatesvc "github.com/johanbrandhorst/trail-tools/internal/services/lactate"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// stages returns lactate test stages at the heart rates and lactates.
func stages(heartRates []int32, lactates []float64) []*lactatev1.LactateStage {
	result := make([]*lactatev1.LactateStage, len(heartRates))
	for i := range heartRates {
		result[i] = lactatev1.LactateStage_builder{HeartRateBpm: heartRates[i], LactateMmolPerLiter: lactates[i]}.Build()
	}
	return result
}

func TestLactateTests(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()

	// A test starts with the resting reading, as in a guided test.
	created, err := e.lactate.CreateLactateTest(ctx, lactatev1.CreateLactateTestRequest_builder{
		Test: lactatev1.LactateTest_builder{Date: "2026-10-10", RestingLactateMmolPerLiter: 1.2}.Build(),
	}.Build())
	if err != nil {
		t.Fatalf("CreateLactateTest: %v", err)
	}
	test := created.GetTest()
	if test.GetLowestLactateMmolPerLiter() != 1.2 || test.GetAerobicThresholdHeartRateBpm() != 0 {
		t.Errorf("got a lowest reading of %v and a threshold of %v, want 1.2 and none", test.GetLowestLactateMmolPerLiter(), test.GetAerobicThresholdHeartRateBpm())
	}

	// Stages are added as the test goes. Lactate dips after the warm-up,
	// then rises past 2 mmol/L between 140 and 150 bpm.
	updated, err := e.lactate.UpdateLactateTest(ctx, lactatev1.UpdateLactateTestRequest_builder{
		Test: lactatev1.LactateTest_builder{
			Id:     test.GetId(),
			Stages: stages([]int32{110, 120, 130, 140, 150, 160}, []float64{1.3, 0.9, 1.1, 1.6, 2.4, 3.4}),
		}.Build(),
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"stages"}},
	}.Build())
	if err != nil {
		t.Fatalf("UpdateLactateTest: %v", err)
	}
	test = updated.GetTest()
	if got := test.GetRestingLactateMmolPerLiter(); got != 1.2 {
		t.Errorf("updating the stages changed the resting reading to %v", got)
	}
	if got := test.GetLowestLactateMmolPerLiter(); got != 0.9 {
		t.Errorf("got a lowest reading of %v, want 0.9", got)
	}
	// 2 mmol/L is halfway from 1.6 at 140 bpm to 2.4 at 150 bpm.
	if got := test.GetAerobicThresholdHeartRateBpm(); got != 145 {
		t.Errorf("got an aerobic threshold of %v bpm, want 145", got)
	}
	want := map[lactatev1.ThresholdDefinition][2]float64{
		lactatev1.ThresholdDefinition_THRESHOLD_DEFINITION_FIXED_2_MMOL:   {2, 145},
		lactatev1.ThresholdDefinition_THRESHOLD_DEFINITION_FIXED_2_5_MMOL: {2.5, 151},
		// 1.9 mmol/L, from 1.6 at 140 bpm to 2.4 at 150 bpm.
		lactatev1.ThresholdDefinition_THRESHOLD_DEFINITION_LOWEST_PLUS_1_MMOL: {1.9, 144},
	}
	if len(test.GetThresholdEstimates()) != len(want) {
		t.Fatalf("got estimates %v, want %d", test.GetThresholdEstimates(), len(want))
	}
	for _, estimate := range test.GetThresholdEstimates() {
		w := want[estimate.GetDefinition()]
		if estimate.GetLactateMmolPerLiter() != w[0] || float64(estimate.GetHeartRateBpm()) != w[1] {
			t.Errorf("got estimate %v, want %v mmol/L at %v bpm", estimate, w[0], w[1])
		}
	}

	// Tests are listed with the latest first.
	if _, err := e.lactate.CreateLactateTest(ctx, lactatev1.CreateLactateTestRequest_builder{
		Test: lactatev1.LactateTest_builder{Date: "2026-07-01", Stages: stages([]int32{120, 130}, []float64{2.1, 2.6})}.Build(),
	}.Build()); err != nil {
		t.Fatalf("CreateLactateTest: %v", err)
	}
	listed, err := e.lactate.ListLactateTests(ctx, &lactatev1.ListLactateTestsRequest{})
	if err != nil {
		t.Fatalf("ListLactateTests: %v", err)
	}
	if tests := listed.GetTests(); len(tests) != 2 || tests[0].GetId() != test.GetId() || tests[1].GetAerobicThresholdHeartRateBpm() != 0 {
		t.Errorf("got tests %v, want the October one first, and no threshold for one starting above it", tests)
	}

	// Invalid stages are rejected.
	for name, s := range map[string][]*lactatev1.LactateStage{
		"heart rate too low":  stages([]int32{20}, []float64{1}),
		"no reading":          stages([]int32{120}, []float64{0}),
		"falling heart rates": stages([]int32{130, 120}, []float64{1, 1}),
	} {
		_, err := e.lactate.UpdateLactateTest(ctx, lactatev1.UpdateLactateTestRequest_builder{
			Test:       lactatev1.LactateTest_builder{Id: test.GetId(), Stages: s}.Build(),
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"stages"}},
		}.Build())
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: got error %v, want invalid argument", name, err)
		}
	}

	// Other users can't see or change the test.
	if _, err := e.otherLactate.GetLactateTest(ctx, lactatev1.GetLactateTestRequest_builder{Id: test.GetId()}.Build()); connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("another user got the test: %v", err)
	}
	if _, err := e.otherLactate.DeleteLactateTest(ctx, lactatev1.DeleteLactateTestRequest_builder{Id: test.GetId()}.Build()); connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("another user deleted the test: %v", err)
	}

	if _, err := e.lactate.DeleteLactateTest(ctx, lactatev1.DeleteLactateTestRequest_builder{Id: test.GetId()}.Build()); err != nil {
		t.Fatalf("DeleteLactateTest: %v", err)
	}
	if _, err := e.lactate.GetLactateTest(ctx, lactatev1.GetLactateTestRequest_builder{Id: test.GetId()}.Build()); connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("got the deleted test: %v", err)
	}
}

func TestLactateTestWorkout(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	get := func(query string) *http.Response {
		t.Helper()
		resp, err := e.srv.Client().Get(e.srv.URL + lactatesvc.WorkoutPath + query)
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		t.Cleanup(func() { _ = resp.Body.Close() })
		return resp
	}

	// Stages from 100 to 120 bpm, each followed by a step to sample in,
	// between a warm-up and a cool-down.
	resp := get("?start=100&step=10&end=120&minutes=5&range=5")
	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Header.Get("Content-Disposition"), "lactate-test.fit") {
		t.Fatalf("got status %d and disposition %q", resp.StatusCode, resp.Header.Get("Content-Disposition"))
	}
	file, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	var ranges [][2]uint64
	for _, m := range fittest.Decode(t, file) {
		if m.Global == 27 {
			names = append(names, m.String(0))
			ranges = append(ranges, [2]uint64{m.Uint(t, 5), m.Uint(t, 6)})
		}
	}
	want := []string{"Warm up", "100 bpm", "Sample", "110 bpm", "Sample", "120 bpm", "Sample", "Cool down"}
	if !slices.Equal(names, want) {
		t.Fatalf("got steps %q, want %q", names, want)
	}
	// Heart rates are offset by 100 in the file.
	if ranges[1] != [2]uint64{195, 205} || ranges[5] != [2]uint64{215, 225} {
		t.Errorf("got ranges %v, want 95-105 and 115-125 bpm", ranges)
	}

	// Invalid stages are rejected.
	for _, query := range []string{"?start=10", "?start=150&end=120", "?start=60&step=2&end=200"} {
		if resp := get(query); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: got status %d, want 400", query, resp.StatusCode)
		}
	}
}
