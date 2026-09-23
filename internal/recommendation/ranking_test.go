package recommendation

import (
	"path/filepath"
	"strings"
	"testing"

	"careerquest/internal/career"
	"careerquest/internal/dataset"
	"careerquest/internal/domain"
	"careerquest/internal/repository"
)

type recommendationStore struct {
	repository.Store
	employee   domain.Employee
	profile    domain.RoleProfile
	events     []domain.Event
	activities []domain.Activity
}

func (s *recommendationStore) Employee(id string) (domain.Employee, bool) {
	return s.employee, id == s.employee.ID
}
func (s *recommendationStore) Employees() []domain.Employee { return []domain.Employee{s.employee} }
func (s *recommendationStore) Profile(role, grade string) (domain.RoleProfile, bool) {
	return s.profile, role == s.profile.Role && grade == s.profile.Grade
}
func (s *recommendationStore) Events() []domain.Event { return s.events }
func (s *recommendationStore) Event(id string) (domain.Event, bool) {
	for _, event := range s.events {
		if event.ID == id {
			return event, true
		}
	}
	return domain.Event{}, false
}
func (s *recommendationStore) ActivitiesForEmployee(id string) []domain.Activity {
	if id == s.employee.ID {
		return s.activities
	}
	return nil
}

func recommendationFixture(t *testing.T) *recommendationStore {
	t.Helper()
	base, err := dataset.Load(filepath.Join("..", "..", "case_1", "career_quest_dataset"))
	if err != nil {
		t.Fatal(err)
	}
	return &recommendationStore{
		Store: base,
		employee: domain.Employee{
			ID: "TEST", Role: "Backend Engineer", Grade: "Junior", LastReviewDate: "2026-09-30",
			CareerGoal: &domain.CareerGoal{TargetRole: "Backend Engineer", TargetGrade: "Middle"},
			Skills:     map[string]int{"SK_API_DESIGN": 2, "SK_SYSTEM_DESIGN": 1, "SK_COMMUNICATION": 1},
		},
		profile: domain.RoleProfile{
			Role: "Backend Engineer", Grade: "Middle",
			RequiredSkills: map[string]int{"SK_API_DESIGN": 3, "SK_SYSTEM_DESIGN": 2, "SK_COMMUNICATION": 2},
			CriticalSkills: []string{"SK_API_DESIGN", "SK_SYSTEM_DESIGN"},
		},
		events: []domain.Event{{
			ID: "TECH", Title: "System Design Fundamentals", Type: "course", Format: "self_paced", DurationHours: 8,
			TargetRoles: []string{"Backend Engineer"}, TargetGrades: []string{"Junior", "Middle"},
			Prerequisites: map[string]int{"SK_API_DESIGN": 2},
			DevelopsSkills: []domain.SkillEffect{
				{SkillID: "SK_API_DESIGN", Gain: 1, MaxLevel: 3},
				{SkillID: "SK_SYSTEM_DESIGN", Gain: 1, MaxLevel: 2},
			},
		}},
	}
}

func recommendationsFor(t *testing.T, store *recommendationStore) []Item {
	t.Helper()
	result, err := New(store, career.New(store)).ForEmployee(store.employee.ID)
	if err != nil {
		t.Fatal(err)
	}
	return result.Recommendations
}

func TestRankingFavorsCriticalProgressOverNarrowRelevance(t *testing.T) {
	store := recommendationFixture(t)
	// Additional content must not push a course closing two critical gaps below
	// a course that only closes one noncritical gap.
	for _, id := range []string{"SK_CLOUD", "SK_CONTAINERS", "SK_CICD", "SK_OBSERVABILITY"} {
		store.events[0].DevelopsSkills = append(store.events[0].DevelopsSkills, domain.SkillEffect{SkillID: id, Gain: 1, MaxLevel: 3})
	}
	narrow := store.events[0]
	narrow.ID, narrow.Title, narrow.DurationHours = "SOFT", "Communication Basics", 1
	narrow.DevelopsSkills = []domain.SkillEffect{{SkillID: "SK_COMMUNICATION", Gain: 1, MaxLevel: 2}}
	store.events = append(store.events, narrow)
	items := recommendationsFor(t, store)
	if len(items) != 2 || items[0].EventID != "TECH" {
		t.Fatalf("expected critical skill progress first, got %+v", items)
	}
}

func TestRecommendationEligibility(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*recommendationStore)
	}{
		{"mandatory", func(s *recommendationStore) { s.events[0].Mandatory = true }},
		{"wrong role", func(s *recommendationStore) { s.events[0].TargetRoles = []string{"Sales Manager"} }},
		{"wrong grade", func(s *recommendationStore) { s.events[0].TargetGrades = []string{"Lead"} }},
		{"missing prerequisite", func(s *recommendationStore) { s.employee.Skills["SK_API_DESIGN"] = 1 }},
		{"past sessions", func(s *recommendationStore) {
			s.events[0].Format, s.events[0].UpcomingSessions = "online", []string{"2026-09-01"}
		}},
		{"no gaps", func(s *recommendationStore) { s.employee.Skills = s.profile.RequiredSkills }},
		{"course below skill level", func(s *recommendationStore) {
			s.events[0].DevelopsSkills = []domain.SkillEffect{{SkillID: "SK_API_DESIGN", Gain: 1, MaxLevel: 2}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := recommendationFixture(t)
			tc.change(store)
			if items := recommendationsFor(t, store); len(items) != 0 {
				t.Fatalf("unexpected recommendations: %+v", items)
			}
		})
	}
	for _, status := range []string{"completed", "in_progress", "planned", "enrolled"} {
		t.Run(status, func(t *testing.T) {
			store := recommendationFixture(t)
			store.activities = []domain.Activity{{EventID: "TECH", Date: "2026-09-01", Status: status}}
			if items := recommendationsFor(t, store); len(items) != 0 {
				t.Fatalf("recommended an activity already %s: %+v", status, items)
			}
			candidates, err := New(store, career.New(store)).CandidatesForEvent("TECH")
			if err != nil || len(candidates) != 0 {
				t.Fatalf("HR candidates include employee already %s: %+v, %v", status, candidates, err)
			}
		})
	}
}

func TestFailedAttemptIsExplainedAndRankedLower(t *testing.T) {
	store := recommendationFixture(t)
	original := recommendationsFor(t, store)[0]
	store.activities = []domain.Activity{{EventID: "TECH", Date: "2026-09-01", Status: "failed"}}
	items := recommendationsFor(t, store)
	if len(items) != 1 || items[0].PreviousStatus != "failed" || items[0].MatchScore >= original.MatchScore {
		t.Fatalf("failed attempt was not accounted for: %+v", items)
	}
}

func TestExplanationIncludesSkillLevelsAndTarget(t *testing.T) {
	store := recommendationFixture(t)
	item := recommendationsFor(t, store)[0]
	for _, text := range []string{"API Design", "2 to 3", "required: 3", "critical", "Backend Engineer Middle", "career goal", "projected"} {
		if !strings.Contains(item.Explanation, text) {
			t.Errorf("explanation missing %q: %s", text, item.Explanation)
		}
	}
	store.employee.CareerGoal = nil
	store.profile.Grade = store.employee.Grade
	item = recommendationsFor(t, store)[0]
	if item.Alignment != "current_role" || item.ReadinessImpact != nil || !strings.Contains(item.Explanation, "current role") {
		t.Fatalf("unexpected recommendation without a goal: %+v", item)
	}
}
