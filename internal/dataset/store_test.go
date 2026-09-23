package dataset

import (
	"path/filepath"
	"testing"

	"careerquest/internal/domain"
)

func TestLoadCareerQuestDataset(t *testing.T) {
	store, err := Load(filepath.Join("..", "..", "case_1", "career_quest_dataset"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := len(store.skills); got != 60 {
		t.Fatalf("skills count = %d, want 60", got)
	}
	if got := len(store.profiles); got != 32 {
		t.Fatalf("profiles count = %d, want 32", got)
	}
	if got := len(store.employees); got != 200 {
		t.Fatalf("employees count = %d, want 200", got)
	}
	if got := len(store.events); got != 40 {
		t.Fatalf("events count = %d, want 40", got)
	}
	var historyCount int
	for _, records := range store.activities {
		historyCount += len(records)
	}
	if historyCount != 2743 {
		t.Fatalf("history count = %d, want 2743", historyCount)
	}
	if store.Meta().AsOfDate != "2026-10-01" {
		t.Fatalf("as_of_date = %q, want 2026-10-01", store.Meta().AsOfDate)
	}
}

func TestCreateAndUpdateEvent(t *testing.T) {
	store, err := Load(filepath.Join("..", "..", "case_1", "career_quest_dataset"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	event := domain.Event{
		Title:            "API Design Lab",
		Description:      "Practical API design exercises.",
		Type:             "workshop",
		Format:           "online",
		DurationHours:    4,
		TargetRoles:      []string{"Backend Engineer"},
		TargetGrades:     []string{"Junior", "Middle"},
		DevelopsSkills:   []domain.SkillEffect{{SkillID: "SK_API_DESIGN", Gain: 1, MaxLevel: 4}},
		Prerequisites:    map[string]int{},
		UpcomingSessions: []string{"2026-11-15"},
		LearningLink:     "https://example.com/api-design",
	}
	created, err := store.CreateEvent(event)
	if err != nil {
		t.Fatalf("CreateEvent() error = %v", err)
	}
	if created.ID == "" || created.LearningLink != event.LearningLink {
		t.Fatalf("unexpected created event: %#v", created)
	}
	created.Title = "Updated API Design Lab"
	updated, err := store.UpdateEvent(created.ID, created)
	if err != nil {
		t.Fatalf("UpdateEvent() error = %v", err)
	}
	if updated.Title != "Updated API Design Lab" {
		t.Fatalf("updated title = %q", updated.Title)
	}
}

func TestEmployeeReturnsDefensiveCopy(t *testing.T) {
	store, err := Load(filepath.Join("..", "..", "case_1", "career_quest_dataset"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	employee, ok := store.Employee("E0001")
	if !ok {
		t.Fatal("employee E0001 not found")
	}
	original := employee.Skills["SK_PYTHON"]
	employee.Skills["SK_PYTHON"] = 99

	again, _ := store.Employee("E0001")
	if again.Skills["SK_PYTHON"] != original {
		t.Fatal("mutating returned employee changed the store")
	}
}
