package navigator

import (
	"path/filepath"
	"strings"
	"testing"

	"careerquest/internal/career"
	"careerquest/internal/dataset"
	"careerquest/internal/recommendation"
)

func TestNextStepExplainsLearningPlan(t *testing.T) {
	store, err := dataset.Load(filepath.Join("..", "..", "case_1", "career_quest_dataset"))
	if err != nil {
		t.Fatal(err)
	}
	careerService := career.New(store)
	service := New(store, careerService, recommendation.New(store, careerService))
	answer, err := service.Answer("E0001", "", "What should I do next?")
	if err != nil {
		t.Fatal(err)
	}
	if answer.LearningPlan == nil || len(answer.LearningPlan.Steps) == 0 || !strings.Contains(answer.Message, answer.LearningPlan.Steps[0].Title) {
		t.Fatalf("expected an explanation of the planned next step: %+v", answer)
	}
	if answer.Mode != "deterministic" {
		t.Fatalf("unexpected mode: %s", answer.Mode)
	}
}
