package career

import (
	"path/filepath"
	"testing"

	"careerquest/internal/dataset"
)

func loadStore(t *testing.T) *dataset.Store {
	t.Helper()
	store, err := dataset.Load(filepath.Join("..", "..", "case_1", "career_quest_dataset"))
	if err != nil {
		t.Fatalf("dataset.Load() error = %v", err)
	}
	return store
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
