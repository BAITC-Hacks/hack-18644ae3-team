package recommendation

import (
	"path/filepath"
	"testing"

	"careerquest/internal/career"
	"careerquest/internal/dataset"
)

func TestRecommendationsAreRankedAndVoluntary(t *testing.T) {
	store, err := dataset.Load(filepath.Join("..", "..", "case_1", "career_quest_dataset"))
	if err != nil {
		t.Fatalf("dataset.Load() error = %v", err)
	}
	service := New(store, career.New(store))
	result, err := service.ForEmployee("E0001")
	if err != nil {
		t.Fatalf("ForEmployee() error = %v", err)
	}
	if len(result.Recommendations) == 0 {
		t.Fatal("expected at least one recommendation")
	}
	for i, item := range result.Recommendations {
		event, _ := store.Event(item.EventID)
		if event.Mandatory {
			t.Fatalf("mandatory event %s was recommended", event.ID)
		}
		if len(item.SkillsCovered) == 0 {
			t.Fatalf("event %s covers no target gaps", event.ID)
		}
		if i > 0 && result.Recommendations[i-1].MatchScore < item.MatchScore {
			t.Fatal("recommendations are not sorted by descending score")
		}
	}
}
