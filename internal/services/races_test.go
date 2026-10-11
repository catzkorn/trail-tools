package services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"
	racesv1 "github.com/catzkorn/trail-tools/internal/gen/races/v1"
	routesv1 "github.com/catzkorn/trail-tools/internal/gen/routes/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestRacesAndRacePlans(t *testing.T) {
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

	// A new plan is seeded with the finish and core contingencies, and most
	// rules fail.
	newPlan, err := e.racePlans.CreateRacePlan(ctx, connect.NewRequest(racesv1.CreateRacePlanRequest_builder{RaceId: raceID}.Build()))
	if err != nil {
		t.Fatalf("CreateRacePlan: %v", err)
	}
	planID := newPlan.Msg.GetPlan().GetId()
	if cps := newPlan.Msg.GetPlan().GetCheckpoints(); len(cps) != 1 || cps[0].GetName() != "Finish" || cps[0].GetCutoffSeconds() != 6*3600 {
		t.Errorf("unexpected seeded checkpoints %v", cps)
	}
	if n := len(newPlan.Msg.GetPlan().GetContingencies()); n != 7 {
		t.Errorf("got %d seeded contingencies, want 7", n)
	}
	if a := newPlan.Msg.GetAnalysis(); a.GetRulesPassed() >= a.GetRulesTotal() || a.GetRulesTotal() < 10 {
		t.Errorf("unexpected rule counts %d/%d", a.GetRulesPassed(), a.GetRulesTotal())
	}

	// Fill in checkpoints and nutrition.
	plan := racesv1.RacePlan_builder{
		Id:    planID,
		Goals: racesv1.RaceGoals_builder{AFinishSeconds: 3 * 3600, BFinishSeconds: 3*3600 + 1800, CFinishSeconds: 4 * 3600}.Build(),
		Checkpoints: []*racesv1.Checkpoint{
			racesv1.Checkpoint_builder{
				Name:                  "CP1",
				DistanceMeters:        11119,
				CutoffSeconds:         5400,
				PlannedSegmentSeconds: 5000,
				PlannedStopSeconds:    300,
				CrewAccess:            true,
				Intake:                []*racesv1.Intake{racesv1.Intake_builder{Product: "Gel", Quantity: 3}.Build()},
			}.Build(),
			racesv1.Checkpoint_builder{
				Name:                  "Finish",
				DistanceMeters:        22239,
				CutoffSeconds:         6 * 3600,
				PlannedSegmentSeconds: 5400,
				Intake:                []*racesv1.Intake{racesv1.Intake_builder{Product: "Mystery bar", Quantity: 1}.Build()},
			}.Build(),
		},
		Nutrition: racesv1.NutritionStrategy_builder{
			CarbsGramsPerHour: 80,
			FluidMlPerHour:    500,
			SodiumMgPerHour:   500,
			Products: []*racesv1.NutritionProduct{
				racesv1.NutritionProduct_builder{Name: "Gel", CarbsGrams: 25, SodiumMg: 100}.Build(),
			},
		}.Build(),
	}.Build()
	updatedPlan, err := e.racePlans.UpdateRacePlan(ctx, connect.NewRequest(racesv1.UpdateRacePlanRequest_builder{
		Plan:       plan,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"goals", "checkpoints", "nutrition"}},
	}.Build()))
	if err != nil {
		t.Fatalf("UpdateRacePlan: %v", err)
	}
	analysis := updatedPlan.Msg.GetAnalysis()
	cp1 := analysis.GetCheckpoints()[0]
	if cp1.GetArrivalClockTime() != "06:23" || cp1.GetCutoffBufferSeconds() != 400 || cp1.GetCarbsGrams() != 75 {
		t.Errorf("unexpected CP1 analysis %v", cp1)
	}
	// The segment elevation comes from the course: 100m per segment.
	if cp1.GetSegmentElevationGainMeters() < 95 || cp1.GetSegmentElevationGainMeters() > 105 {
		t.Errorf("got CP1 gain %v from the course, want ~100", cp1.GetSegmentElevationGainMeters())
	}
	if analysis.GetPlannedFinishSeconds() != 10700 {
		t.Errorf("got finish %d, want 10700", analysis.GetPlannedFinishSeconds())
	}
	rules := map[string]*racesv1.RuleResult{}
	for _, r := range analysis.GetRules() {
		rules[r.GetId()] = r
	}
	for id, want := range map[string]racesv1.RuleStatus{
		"goals":           racesv1.RuleStatus_RULE_STATUS_PASS,
		"checkpoints":     racesv1.RuleStatus_RULE_STATUS_PASS,
		"segment-times":   racesv1.RuleStatus_RULE_STATUS_PASS,
		"cutoff-buffers":  racesv1.RuleStatus_RULE_STATUS_WARN,
		"known-products":  racesv1.RuleStatus_RULE_STATUS_FAIL,
		"segment-fueling": racesv1.RuleStatus_RULE_STATUS_FAIL,
		"night-gear":      racesv1.RuleStatus_RULE_STATUS_FAIL,
		"crew":            racesv1.RuleStatus_RULE_STATUS_PASS,
		"contingencies":   racesv1.RuleStatus_RULE_STATUS_FAIL,
	} {
		if got := rules[id].GetStatus(); got != want {
			t.Errorf("rule %s: got %v (%s), want %v", id, got, rules[id].GetMessage(), want)
		}
	}
	if !strings.Contains(rules["known-products"].GetMessage(), "Mystery bar") {
		t.Errorf("unexpected known-products message %q", rules["known-products"].GetMessage())
	}
	// Other sections were left alone.
	if n := len(updatedPlan.Msg.GetPlan().GetContingencies()); n != 7 {
		t.Errorf("contingencies were changed, got %d", n)
	}

	_, err = e.racePlans.UpdateRacePlan(ctx, connect.NewRequest(racesv1.UpdateRacePlanRequest_builder{
		Plan:       racesv1.RacePlan_builder{Id: planID}.Build(),
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"share_url"}},
	}.Build()))
	wantCode(t, err, connect.CodeInvalidArgument)

	// Share the plan, view it anonymously, then stop sharing.
	shared, err := e.racePlans.ShareRacePlan(ctx, connect.NewRequest(racesv1.ShareRacePlanRequest_builder{Id: planID}.Build()))
	if err != nil {
		t.Fatalf("ShareRacePlan: %v", err)
	}
	shareURL := shared.Msg.GetShareUrl()
	token, ok := strings.CutPrefix(shareURL, "https://trail.example/share/race-plans/")
	if !ok || token == "" {
		t.Fatalf("unexpected share URL %q", shareURL)
	}
	again, err := e.racePlans.ShareRacePlan(ctx, connect.NewRequest(racesv1.ShareRacePlanRequest_builder{Id: planID}.Build()))
	if err != nil || again.Msg.GetShareUrl() != shareURL {
		t.Errorf("sharing again should return the same URL: %v %v", again, err)
	}
	view, err := e.anonPlans.GetSharedRacePlan(ctx, connect.NewRequest(racesv1.GetSharedRacePlanRequest_builder{Token: token}.Build()))
	if err != nil {
		t.Fatalf("GetSharedRacePlan: %v", err)
	}
	if view.Msg.GetRace().GetName() != "Mountain Ultra" || len(view.Msg.GetCourse()) == 0 || view.Msg.GetRace().GetCourseRouteId() != "" {
		t.Errorf("unexpected shared plan %v", view.Msg.GetRace())
	}
	if _, err := e.racePlans.UnshareRacePlan(ctx, connect.NewRequest(racesv1.UnshareRacePlanRequest_builder{Id: planID}.Build())); err != nil {
		t.Fatalf("UnshareRacePlan: %v", err)
	}
	_, err = e.anonPlans.GetSharedRacePlan(ctx, connect.NewRequest(racesv1.GetSharedRacePlanRequest_builder{Token: token}.Build()))
	wantCode(t, err, connect.CodeNotFound)

	// Other users can't see the plan, and anonymous users can't use the
	// authenticated API.
	_, err = e.anonPlans.GetRacePlan(ctx, connect.NewRequest(racesv1.GetRacePlanRequest_builder{Id: planID}.Build()))
	wantCode(t, err, connect.CodeUnauthenticated)

	list, err := e.racePlans.ListRacePlans(ctx, connect.NewRequest(racesv1.ListRacePlansRequest_builder{}.Build()))
	if err != nil {
		t.Fatalf("ListRacePlans: %v", err)
	}
	if len(list.Msg.GetPlans()) != 1 || list.Msg.GetPlans()[0].GetShared() {
		t.Errorf("unexpected plans %v", list.Msg.GetPlans())
	}

	// Deleting the race deletes its plans.
	if _, err := e.races.DeleteRace(ctx, connect.NewRequest(racesv1.DeleteRaceRequest_builder{Id: raceID}.Build())); err != nil {
		t.Fatalf("DeleteRace: %v", err)
	}
	_, err = e.racePlans.GetRacePlan(ctx, connect.NewRequest(racesv1.GetRacePlanRequest_builder{Id: planID}.Build()))
	wantCode(t, err, connect.CodeNotFound)
}

func TestMCP(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()

	// Unauthenticated requests are rejected.
	resp, err := e.srv.Client().Post(e.srv.URL+"/mcp", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized || !strings.HasPrefix(resp.Header.Get("WWW-Authenticate"), "Bearer") {
		t.Errorf("got status %d, want 401 with a challenge", resp.StatusCode)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   e.srv.URL + "/mcp",
		HTTPClient: &http.Client{Transport: &bearerTransport{token: e.token}},
	}, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	for _, want := range []string{"CreateRace", "ListRaces", "PreviewRoute", "GetRouteForecast"} {
		if !slices.Contains(names, want) {
			t.Errorf("missing tool %s in %v", want, names)
		}
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "CreateRace",
		Arguments: map[string]any{
			"race": map[string]any{"name": "Agent race", "date": "2027-01-01", "distanceMeters": 50000},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("CreateRace failed: %v", res.Content[0].(*mcp.TextContent).Text)
	}
	// The race was created for the token's user.
	races, err := e.races.ListRaces(ctx, connect.NewRequest(racesv1.ListRacesRequest_builder{}.Build()))
	if err != nil {
		t.Fatal(err)
	}
	if len(races.Msg.GetRaces()) != 1 || races.Msg.GetRaces()[0].GetName() != "Agent race" {
		t.Errorf("unexpected races %v", races.Msg.GetRaces())
	}
	var out struct {
		Race struct {
			ID string `json:"id"`
		} `json:"race"`
	}
	if err := json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out); err != nil || out.Race.ID == "" {
		t.Errorf("unexpected tool output %v: %v", res.Content, err)
	}

	// Errors are returned to the agent so it can correct itself.
	res, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "CreateRace",
		Arguments: map[string]any{"race": map[string]any{"name": "No date"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(*mcp.TextContent).Text, "invalid_argument") {
		t.Errorf("expected an invalid argument tool error, got %v", res.Content)
	}
}

type bearerTransport struct {
	token string
}

func (b *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(req)
}
