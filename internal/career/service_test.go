package career

import (
	"path/filepath"
	"testing"

	"careerquest/internal/dataset"
	"careerquest/internal/domain"
)

func loadStore(t *testing.T) *dataset.Store {
	t.Helper()
	store, err := dataset.Load(filepath.Join("..", "..", "case_1", "career_quest_dataset"))
	if err != nil {
		t.Fatalf("dataset.Load() error = %v", err)
	}
	return store
}

func TestProjectEventPreservesSkillsAboveCourseCap(t *testing.T) {
	levels := map[string]int{"advanced": 5, "learning": 1, "capped": 2}
	event := domain.Event{DevelopsSkills: []domain.SkillEffect{
		{SkillID: "advanced", Gain: 1, MaxLevel: 3},
		{SkillID: "learning", Gain: 1, MaxLevel: 3},
		{SkillID: "capped", Gain: 2, MaxLevel: 3},
	}}
	projected := ProjectEvent(levels, event)
	for skill, want := range map[string]int{"advanced": 5, "learning": 2, "capped": 3} {
		if projected[skill] != want {
			t.Errorf("projected %s = %d, want %d", skill, projected[skill], want)
		}
	}
	if levels["learning"] != 1 || levels["capped"] != 2 {
		t.Fatal("projection changed the employee's assessed skills")
	}
}

func TestEffectiveSkillsIncludePostReviewCompletions(t *testing.T) {
	store := loadStore(t)
	service := New(store)
	employee, ok := store.Employee("E0004")
	if !ok {
		t.Fatal("employee E0004 not found")
	}
	before := employee.Skills["SK_ML_BASICS"]
	after := service.EffectiveSkills(employee)["SK_ML_BASICS"]
	if after <= before {
		t.Fatalf("effective SK_ML_BASICS = %d, want greater than assessed level %d", after, before)
	}
}

func TestAssessmentUsesCareerGoal(t *testing.T) {
	store := loadStore(t)
	assessment, err := New(store).Assess("E0004")
	if err != nil {
		t.Fatalf("Assess() error = %v", err)
	}
	if assessment.Basis != "career_goal" || assessment.TargetRole != "Product Manager" || assessment.TargetGrade != "Middle" {
		t.Fatalf("unexpected assessment target: %#v", assessment)
	}
	if assessment.ReadinessPercent == nil {
		t.Fatal("goal-based assessment has nil readiness")
	}
	if len(assessment.Gaps) == 0 {
		t.Fatal("expected target skill gaps")
	}
}
