package navigator

import (
	"fmt"
	"strings"

	"careerquest/internal/career"
	"careerquest/internal/recommendation"
	"careerquest/internal/repository"
)

type Service struct {
	store          repository.Store
	career         *career.Service
	recommendation *recommendation.Service
}

func New(store repository.Store, careerService *career.Service, recommendationService *recommendation.Service) *Service {
	return &Service{store: store, career: careerService, recommendation: recommendationService}
}

type Answer struct {
	Mode           string                       `json:"mode"`
	EmployeeID     string                       `json:"employee_id"`
	EventID        string                       `json:"event_id,omitempty"`
	Message        string                       `json:"message"`
	Recommendation *recommendation.Item         `json:"recommendation,omitempty"`
	LearningPlan   *recommendation.LearningPlan `json:"learning_plan,omitempty"`
}

func (s *Service) Answer(employeeID, eventID, question string) (Answer, error) {
	employee, ok := s.store.Employee(employeeID)
	if !ok {
		return Answer{}, fmt.Errorf("employee %q not found", employeeID)
	}
	result, err := s.recommendation.ForEmployee(employeeID)
	if err != nil {
		return Answer{}, err
	}
	answer := Answer{Mode: "deterministic", EmployeeID: employeeID}
	var selected *recommendation.Item
	if eventID != "" {
		for i := range result.Recommendations {
			if result.Recommendations[i].EventID == eventID {
				selected = &result.Recommendations[i]
				break
			}
		}
		if selected == nil {
			return Answer{}, fmt.Errorf("event is not currently recommended for this employee")
		}
	}

	assessment, err := s.career.AssessEmployee(employee)
	if err != nil {
		return Answer{}, err
	}
	lowerQuestion := strings.ToLower(question)
	if selected == nil && containsAny(lowerQuestion, "block", "missing", "promotion", "skill gap") {
		if employee.CareerGoal == nil {
			answer.Message = "Set a career goal first. Career Quest needs a target role and grade before it can identify promotion blockers."
			return answer, nil
		}
		parts := make([]string, 0, 4)
		for _, gap := range assessment.Gaps {
			if len(parts) == 4 {
				break
			}
			if gap.Critical || len(parts) < 2 {
				parts = append(parts, fmt.Sprintf("%s (%d/%d)", gap.SkillName, gap.CurrentLevel, gap.RequiredLevel))
			}
		}
		answer.Message = fmt.Sprintf("Your main gaps for %s %s are %s.", assessment.TargetRole, assessment.TargetGrade, strings.Join(parts, ", "))
		if assessment.ReadinessPercent != nil {
			answer.Message += fmt.Sprintf(" Current career readiness is %.1f%%.", *assessment.ReadinessPercent)
		}
		return answer, nil
	}
	if selected == nil && containsAny(lowerQuestion, "change my career goal", "change goal", "different goal") {
		answer.Message = "Changing your career goal recalculates the required skill profile, readiness percentage, blockers, and recommendation ranking. Your assessed skills and activity history stay the same."
		return answer, nil
	}
	if selected == nil && result.LearningPlan != nil && containsAny(lowerQuestion, "next", "plan", "path", "start") {
		plan := result.LearningPlan
		steps := make([]string, 0, len(plan.Steps))
		for index, step := range plan.Steps {
			steps = append(steps, fmt.Sprintf("%d. %s", index+1, step.Title))
		}
		answer.EventID = plan.Steps[0].EventID
		answer.LearningPlan = plan
		answer.Message = fmt.Sprintf("Suggested learning order: %s. Start with %s. %s Total study time: %.1f hours.", strings.Join(steps, "; "), plan.Steps[0].Title, plan.Steps[0].Explanation, plan.TotalDurationHours)
		if plan.ReadinessAfterPercent != nil {
			answer.Message += fmt.Sprintf(" Completing and passing all steps is projected to increase readiness from %.1f%% to %.1f%%.", *plan.ReadinessBeforePercent, *plan.ReadinessAfterPercent)
		}
		return answer, nil
	}
	if selected == nil && len(result.Recommendations) > 0 {
		selected = &result.Recommendations[0]
	}
	if selected == nil {
		answer.Message = "No eligible skill-building activity currently matches your gaps. This can happen when prerequisites are not met, matching activities are already completed, or no upcoming session is available."
		return answer, nil
	}
	answer.EventID = selected.EventID
	answer.Message = selected.Explanation
	answer.Recommendation = selected
	return answer, nil
}

func containsAny(value string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}
