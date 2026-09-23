package assessment

import (
	"errors"
	"strings"

	"careerquest/internal/repository"
)

// Service owns course-result rules while the repository supplies the atomic
// transaction that records the result and applies skill rewards.
type Service struct {
	manager repository.AssessmentManager
}

func New(store repository.Store) *Service {
	manager, _ := store.(repository.AssessmentManager)
	return &Service{manager: manager}
}

func (s *Service) Participants(eventID string) ([]repository.Participant, error) {
	if s.manager == nil {
		return nil, errors.New("participant management is unavailable")
	}
	return s.manager.EventParticipants(eventID)
}

func (s *Service) Assess(enrollmentID int64, assessorUserID string, input repository.AssessmentInput) (repository.AssessmentResult, error) {
	if s.manager == nil {
		return repository.AssessmentResult{}, errors.New("assessment management is unavailable")
	}
	input.Result = strings.ToUpper(strings.TrimSpace(input.Result))
	input.Feedback = strings.TrimSpace(input.Feedback)
	if input.Result != "PASSED" && input.Result != "FAILED" {
		return repository.AssessmentResult{}, errors.New("result must be PASSED or FAILED")
	}
	if input.Score != nil && (*input.Score < 0 || *input.Score > 100) {
		return repository.AssessmentResult{}, errors.New("score must be between 0 and 100")
	}
	return s.manager.AssessEnrollment(enrollmentID, assessorUserID, input)
}
