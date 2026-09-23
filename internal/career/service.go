package career

import (
	"fmt"
	"math"
	"sort"

	"careerquest/internal/domain"
	"careerquest/internal/repository"
)

type Service struct {
	store repository.Store
}

func New(store repository.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Assess(employeeID string) (domain.Assessment, error) {
	employee, ok := s.store.Employee(employeeID)
	if !ok {
		return domain.Assessment{}, fmt.Errorf("employee %q not found", employeeID)
	}
	return s.AssessEmployee(employee)
}

func (s *Service) AssessEmployee(employee domain.Employee) (domain.Assessment, error) {
	basis := "current_role"
	targetRole, targetGrade := employee.Role, employee.Grade
	if employee.CareerGoal != nil {
		basis = "career_goal"
		targetRole = employee.CareerGoal.TargetRole
		targetGrade = employee.CareerGoal.TargetGrade
	}
	profile, ok := s.store.Profile(targetRole, targetGrade)
	if !ok {
		return domain.Assessment{}, fmt.Errorf("role profile %s/%s not found", targetRole, targetGrade)
	}

	effective := s.EffectiveSkills(employee)
	gaps := s.Gaps(profile, effective)
	var readiness *float64
	if employee.CareerGoal != nil {
		value := round1(Readiness(profile, effective))
		readiness = &value
	}
	return domain.Assessment{
		EmployeeID:       employee.ID,
		Basis:            basis,
		TargetRole:       targetRole,
		TargetGrade:      targetGrade,
		EffectiveSkills:  effective,
		Gaps:             gaps,
		ReadinessPercent: readiness,
	}, nil
}

// EffectiveSkills applies completed event effects that occurred after the
// employee's last assessment. Activity dates and review dates are ISO dates,
// so lexical comparison preserves chronological order.
func (s *Service) EffectiveSkills(employee domain.Employee) map[string]int {
	levels := cloneLevels(employee.Skills)
	for _, activity := range s.store.ActivitiesForEmployee(employee.ID) {
		if activity.Status != "completed" || activity.SkillRewardsApplied || activity.Date <= employee.LastReviewDate {
			continue
		}
		event, ok := s.store.Event(activity.EventID)
		if !ok {
			continue
		}
		ApplyEvent(levels, event)
	}
	return levels
}

func (s *Service) Gaps(profile domain.RoleProfile, levels map[string]int) []domain.SkillGap {
	critical := make(map[string]bool, len(profile.CriticalSkills))
	for _, skillID := range profile.CriticalSkills {
		critical[skillID] = true
	}
	result := make([]domain.SkillGap, 0)
	for skillID, required := range profile.RequiredSkills {
		current := levels[skillID]
		if current >= required {
			continue
		}
		skill, _ := s.store.Skill(skillID)
		result = append(result, domain.SkillGap{
			SkillID:       skillID,
			SkillName:     skill.Name,
			CurrentLevel:  current,
			RequiredLevel: required,
			MissingLevels: required - current,
			Critical:      critical[skillID],
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Critical != result[j].Critical {
			return result[i].Critical
		}
		if result[i].MissingLevels != result[j].MissingLevels {
			return result[i].MissingLevels > result[j].MissingLevels
		}
		return result[i].SkillID < result[j].SkillID
	})
	return result
}

// Readiness returns weighted requirement fulfillment as a percentage.
// Critical requirements have twice the weight of other requirements.
func Readiness(profile domain.RoleProfile, levels map[string]int) float64 {
	critical := make(map[string]bool, len(profile.CriticalSkills))
	for _, skillID := range profile.CriticalSkills {
		critical[skillID] = true
	}
	var achieved, total float64
	for skillID, required := range profile.RequiredSkills {
		if required <= 0 {
			continue
		}
		weight := 1.0
		if critical[skillID] {
			weight = 2
		}
		progress := math.Min(float64(levels[skillID])/float64(required), 1)
		achieved += weight * progress
		total += weight
	}
	if total == 0 {
		return 100
	}
	return 100 * achieved / total
}

func ApplyEvent(levels map[string]int, event domain.Event) {
	for _, effect := range event.DevelopsSkills {
		current := levels[effect.SkillID]
		levels[effect.SkillID] = max(current, min(current+effect.Gain, effect.MaxLevel))
	}
}

func ProjectEvent(levels map[string]int, event domain.Event) map[string]int {
	projected := cloneLevels(levels)
	ApplyEvent(projected, event)
	return projected
}

func cloneLevels(src map[string]int) map[string]int {
	dst := make(map[string]int, len(src))
	for skillID, level := range src {
		dst[skillID] = level
	}
	return dst
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}
