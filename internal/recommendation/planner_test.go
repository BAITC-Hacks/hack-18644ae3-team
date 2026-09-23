package recommendation

import (
	"strings"
	"testing"

	"careerquest/internal/career"
	"careerquest/internal/domain"
)

func prerequisiteFixture(t *testing.T) *recommendationStore {
	t.Helper()
	store := recommendationFixture(t)
	advanced := store.events[0]
	advanced.ID, advanced.Title = "ADVANCED", "Advanced Architecture"
	advanced.Prerequisites = map[string]int{"SK_SQL": 1}
	foundation := advanced
	foundation.ID, foundation.Title, foundation.DurationHours = "FOUNDATION", "SQL Foundations", 2
	foundation.Prerequisites = nil
	foundation.DevelopsSkills = []domain.SkillEffect{{SkillID: "SK_SQL", Gain: 1, MaxLevel: 1}}
	store.events = []domain.Event{advanced, foundation}
	return store
}

func planFor(t *testing.T, store *recommendationStore) *LearningPlan {
	t.Helper()
	result, err := New(store, career.New(store)).ForEmployee(store.employee.ID)
	if err != nil {
		t.Fatal(err)
	}
	return result.LearningPlan
}

func TestPlanFindsFoundationOutsideTargetSkills(t *testing.T) {
	store := prerequisiteFixture(t)
	if items := recommendationsFor(t, store); len(items) != 0 {
		t.Fatalf("expected no immediately eligible gap-closing course, got %+v", items)
	}
	plan := planFor(t, store)
	if plan == nil || len(plan.Steps) != 2 || plan.Steps[0].EventID != "FOUNDATION" || plan.Steps[1].EventID != "ADVANCED" {
		t.Fatalf("expected foundation followed by unlocked course: %+v", plan)
	}
	if !strings.Contains(plan.Steps[0].Explanation, "unlocks the prerequisites for Advanced Architecture") {
		t.Fatalf("missing prerequisite explanation: %s", plan.Steps[0].Explanation)
	}
	if *plan.ReadinessAfterPercent <= *plan.ReadinessBeforePercent || plan.RemainingGapCount != 1 {
		t.Fatalf("unexpected projected outcome: %+v", plan)
	}
	if store.employee.Skills["SK_SQL"] != 0 || store.employee.Skills["SK_API_DESIGN"] != 2 {
		t.Fatal("planning changed the employee's actual skill levels")
	}
}

func TestPlanFindsThreeCoursePrerequisiteChain(t *testing.T) {
	store := prerequisiteFixture(t)
	store.events[0].Prerequisites = map[string]int{"SK_CLOUD": 1}
	bridge := store.events[1]
	bridge.ID, bridge.Title = "BRIDGE", "Cloud Basics"
	bridge.Prerequisites = map[string]int{"SK_SQL": 1}
	bridge.DevelopsSkills = []domain.SkillEffect{{SkillID: "SK_CLOUD", Gain: 1, MaxLevel: 1}}
	store.events = append(store.events, bridge)
	plan := planFor(t, store)
	if plan == nil || len(plan.Steps) != 3 {
		t.Fatalf("expected three steps: %+v", plan)
	}
	for i, want := range []string{"FOUNDATION", "BRIDGE", "ADVANCED"} {
		if plan.Steps[i].EventID != want {
			t.Errorf("step %d = %s, want %s", i, plan.Steps[i].EventID, want)
		}
	}
}

func TestPlanPrefersLessTimeForSameProgress(t *testing.T) {
	store := recommendationFixture(t)
	short := store.events[0]
	short.ID, short.DurationHours = "SHORT", 2
	store.events = append(store.events, short)
	plan := planFor(t, store)
	if plan == nil || len(plan.Steps) != 1 || plan.Steps[0].EventID != "SHORT" || plan.TotalDurationHours != 2 {
		t.Fatalf("expected shortest equally effective path: %+v", plan)
	}
}

func TestPlanOrdersEquallyUsefulCoursesForEarlierProgress(t *testing.T) {
	store := recommendationFixture(t)
	store.events[0].ID = "Z_CRITICAL"
	soft := store.events[0]
	soft.ID = "A_COMMUNICATION"
	soft.DevelopsSkills = []domain.SkillEffect{{SkillID: "SK_COMMUNICATION", Gain: 1, MaxLevel: 2}}
	store.events = append(store.events, soft)
	plan := planFor(t, store)
	if plan == nil || len(plan.Steps) != 2 || plan.Steps[0].EventID != "Z_CRITICAL" {
		t.Fatalf("expected greater critical progress earlier in the path: %+v", plan)
	}
}

func TestPlanExcludesIneligibleSteps(t *testing.T) {
	for _, status := range []string{"completed", "planned", "enrolled", "in_progress"} {
		t.Run(status, func(t *testing.T) {
			store := prerequisiteFixture(t)
			store.activities = []domain.Activity{{EventID: "FOUNDATION", Status: status, Date: "2026-09-01"}}
			if plan := planFor(t, store); plan != nil {
				t.Fatalf("used unavailable foundation: %+v", plan)
			}
		})
	}
	for _, change := range []func(*recommendationStore){
		func(s *recommendationStore) { s.events[1].Mandatory = true },
		func(s *recommendationStore) { s.events[1].TargetRoles = []string{"Sales Manager"} },
		func(s *recommendationStore) { s.events[1].TargetGrades = []string{"Lead"} },
	} {
		store := prerequisiteFixture(t)
		change(store)
		if plan := planFor(t, store); plan != nil {
			t.Fatalf("used ineligible foundation: %+v", plan)
		}
	}
}

func TestPlanDoesNotScheduleAdvancedSessionBeforeFoundation(t *testing.T) {
	store := prerequisiteFixture(t)
	store.events[0].Format, store.events[0].UpcomingSessions = "online", []string{"2026-10-10"}
	store.events[1].Format, store.events[1].UpcomingSessions = "online", []string{"2026-10-20"}
	if plan := planFor(t, store); plan != nil {
		t.Fatalf("advanced session precedes its prerequisite: %+v", plan)
	}
	store.events[0].UpcomingSessions = append(store.events[0].UpcomingSessions, "2026-10-30")
	if plan := planFor(t, store); plan == nil || len(plan.Steps) != 2 {
		t.Fatalf("later advanced session should make the path possible: %+v", plan)
	}
}

func TestPlanDoesNotInventProgressForSatisfiedGoal(t *testing.T) {
	store := recommendationFixture(t)
	store.employee.Skills = store.profile.RequiredSkills
	if plan := planFor(t, store); plan != nil {
		t.Fatalf("expected no plan for an already satisfied goal: %+v", plan)
	}
}

func TestPersistedSkillRewardsAreNotAppliedTwice(t *testing.T) {
	store := prerequisiteFixture(t)
	store.employee.Skills["SK_SQL"] = 1
	store.events[1].DevelopsSkills[0].MaxLevel = 3
	store.activities = []domain.Activity{{EventID: "FOUNDATION", Status: "completed", Date: "2026-10-01", SkillRewardsApplied: true}}
	levels := career.New(store).EffectiveSkills(store.employee)
	if levels["SK_SQL"] != 1 {
		t.Fatalf("completion reward counted twice: SQL = %d", levels["SK_SQL"])
	}
}
