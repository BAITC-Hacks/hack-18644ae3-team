package recommendation

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"careerquest/internal/career"
	"careerquest/internal/domain"
	"careerquest/internal/repository"
)

const recurringVoluntaryEventID = "EV_036"

type SkillImpact struct {
	SkillID        string `json:"skill_id"`
	SkillName      string `json:"skill_name"`
	BeforeLevel    int    `json:"before_level"`
	ProjectedLevel int    `json:"projected_level"`
	RequiredLevel  int    `json:"required_level"`
	Critical       bool   `json:"critical"`
}

type Item struct {
	EventID                string        `json:"event_id"`
	Title                  string        `json:"title"`
	Type                   string        `json:"type"`
	Format                 string        `json:"format"`
	DurationHours          float64       `json:"duration_hours"`
	LearningLink           string        `json:"learning_link,omitempty"`
	MatchScore             float64       `json:"match_score"`
	Alignment              string        `json:"alignment"`
	SkillsCovered          []SkillImpact `json:"skills_covered"`
	ReadinessBeforePercent *float64      `json:"readiness_before_percent,omitempty"`
	ReadinessAfterPercent  *float64      `json:"readiness_after_percent,omitempty"`
	ReadinessImpact        *float64      `json:"readiness_impact,omitempty"`
	UpcomingSessions       []string      `json:"upcoming_sessions"`
	PreviousStatus         string        `json:"previous_status,omitempty"`
	Explanation            string        `json:"explanation"`
}

type Result struct {
	EmployeeID      string        `json:"employee_id"`
	Basis           string        `json:"basis"`
	TargetRole      string        `json:"target_role"`
	TargetGrade     string        `json:"target_grade"`
	AsOfDate        string        `json:"as_of_date"`
	Recommendations []Item        `json:"recommendations"`
	LearningPlan    *LearningPlan `json:"learning_plan,omitempty"`
}

type Candidate struct {
	EmployeeID      string   `json:"employee_id"`
	FullName        string   `json:"full_name"`
	Role            string   `json:"role"`
	Grade           string   `json:"grade"`
	Department      string   `json:"department"`
	MatchScore      float64  `json:"match_score"`
	ReadinessImpact *float64 `json:"readiness_impact,omitempty"`
	Alignment       string   `json:"alignment"`
}

type Service struct {
	store  repository.Store
	career *career.Service
}

func New(store repository.Store, careerService *career.Service) *Service {
	return &Service{store: store, career: careerService}
}

func (s *Service) CandidatesForEvent(eventID string) ([]Candidate, error) {
	if _, ok := s.store.Event(eventID); !ok {
		return nil, fmt.Errorf("event %q not found", eventID)
	}
	candidates := make([]Candidate, 0)
	for _, employee := range s.store.Employees() {
		result, err := s.forEmployee(employee.ID, false)
		if err != nil {
			return nil, err
		}
		for _, item := range result.Recommendations {
			if item.EventID != eventID {
				continue
			}
			candidates = append(candidates, Candidate{
				EmployeeID:      employee.ID,
				FullName:        employee.FullName,
				Role:            employee.Role,
				Grade:           employee.Grade,
				Department:      employee.Department,
				MatchScore:      item.MatchScore,
				ReadinessImpact: item.ReadinessImpact,
				Alignment:       item.Alignment,
			})
			break
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].MatchScore != candidates[j].MatchScore {
			return candidates[i].MatchScore > candidates[j].MatchScore
		}
		return candidates[i].EmployeeID < candidates[j].EmployeeID
	})
	return candidates, nil
}

func (s *Service) ForEmployee(employeeID string) (Result, error) {
	return s.forEmployee(employeeID, true)
}

func (s *Service) forEmployee(employeeID string, includePlan bool) (Result, error) {
	employee, ok := s.store.Employee(employeeID)
	if !ok {
		return Result{}, fmt.Errorf("employee %q not found", employeeID)
	}
	assessment, err := s.career.AssessEmployee(employee)
	if err != nil {
		return Result{}, err
	}
	profile, ok := s.store.Profile(assessment.TargetRole, assessment.TargetGrade)
	if !ok {
		return Result{}, fmt.Errorf("role profile %s/%s not found", assessment.TargetRole, assessment.TargetGrade)
	}

	critical := make(map[string]bool, len(profile.CriticalSkills))
	for _, skillID := range profile.CriticalSkills {
		critical[skillID] = true
	}
	gaps := make(map[string]int, len(assessment.Gaps))
	var totalWeightedGap float64
	for _, gap := range assessment.Gaps {
		gaps[gap.SkillID] = gap.MissingLevels
		totalWeightedGap += float64(gap.MissingLevels) * skillWeight(gap.Critical)
	}

	history := s.store.ActivitiesForEmployee(employee.ID)
	completed := make(map[string]bool)
	latest := make(map[string]domain.Activity)
	for _, activity := range history {
		if activity.Status == "completed" {
			completed[activity.EventID] = true
		}
		previous, exists := latest[activity.EventID]
		if !exists || activity.Date > previous.Date || (activity.Date == previous.Date && activity.RecordID > previous.RecordID) {
			latest[activity.EventID] = activity
		}
	}

	items := make([]Item, 0)
	planEvents := make([]domain.Event, 0)
	for _, event := range s.store.Events() {
		if event.Mandatory || len(event.DevelopsSkills) == 0 {
			continue
		}
		if completed[event.ID] && event.ID != recurringVoluntaryEventID {
			continue
		}
		previous := latest[event.ID]
		if previous.Status == "in_progress" || previous.Status == "planned" || previous.Status == "enrolled" {
			continue
		}
		alignment, alignmentValue, eligible := alignment(employee, event)
		if !eligible {
			continue
		}
		upcoming, available := availability(event, s.store.Meta().AsOfDate)
		if !available {
			continue
		}
		planEvents = append(planEvents, event)
		if !meetsPrerequisites(assessment.EffectiveSkills, event.Prerequisites) {
			continue
		}

		impacts := make([]SkillImpact, 0)
		var usefulWeighted, offeredWeighted float64
		for _, effect := range event.DevelopsSkills {
			before := assessment.EffectiveSkills[effect.SkillID]
			possible := min(effect.Gain, max(effect.MaxLevel-before, 0))
			if possible == 0 {
				continue
			}
			weight := skillWeight(critical[effect.SkillID])
			offeredWeighted += float64(possible) * weight
			missing, isGap := gaps[effect.SkillID]
			if !isGap {
				continue
			}
			useful := min(possible, missing)
			if useful == 0 {
				continue
			}
			usefulWeighted += float64(useful) * weight
			skill, _ := s.store.Skill(effect.SkillID)
			impacts = append(impacts, SkillImpact{
				SkillID:        effect.SkillID,
				SkillName:      skill.Name,
				BeforeLevel:    before,
				ProjectedLevel: before + possible,
				RequiredLevel:  profile.RequiredSkills[effect.SkillID],
				Critical:       critical[effect.SkillID],
			})
		}
		if usefulWeighted == 0 || offeredWeighted == 0 {
			continue
		}
		sort.Slice(impacts, func(i, j int) bool {
			if impacts[i].Critical != impacts[j].Critical {
				return impacts[i].Critical
			}
			return impacts[i].SkillID < impacts[j].SkillID
		})

		relevance := usefulWeighted / offeredWeighted
		coverage := 0.0
		if totalWeightedGap > 0 {
			coverage = usefulWeighted / totalWeightedGap
		}
		// Prioritize how much of the employee's weighted skill gap is closed.
		// Relevance alone favors narrow courses even when they offer little progress.
		score := 70*coverage + 20*relevance + 10*alignmentValue
		previousStatus := ""
		if isUnsuccessful(previous.Status) {
			previousStatus = previous.Status
			score -= 10
		}
		score = round1(math.Max(0, math.Min(score, 100)))

		item := Item{
			EventID:          event.ID,
			Title:            event.Title,
			Type:             event.Type,
			Format:           event.Format,
			DurationHours:    event.DurationHours,
			LearningLink:     event.LearningLink,
			MatchScore:       score,
			Alignment:        alignment,
			SkillsCovered:    impacts,
			UpcomingSessions: upcoming,
			PreviousStatus:   previousStatus,
		}
		if employee.CareerGoal != nil {
			before := round1(career.Readiness(profile, assessment.EffectiveSkills))
			after := round1(career.Readiness(profile, career.ProjectEvent(assessment.EffectiveSkills, event)))
			impact := round1(after - before)
			item.ReadinessBeforePercent = &before
			item.ReadinessAfterPercent = &after
			item.ReadinessImpact = &impact
		}
		item.Explanation = explain(item, assessment)
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].MatchScore != items[j].MatchScore {
			return items[i].MatchScore > items[j].MatchScore
		}
		leftImpact, rightImpact := pointerValue(items[i].ReadinessImpact), pointerValue(items[j].ReadinessImpact)
		if leftImpact != rightImpact {
			return leftImpact > rightImpact
		}
		if items[i].DurationHours != items[j].DurationHours {
			return items[i].DurationHours < items[j].DurationHours
		}
		return items[i].EventID < items[j].EventID
	})

	result := Result{
		EmployeeID:      employee.ID,
		Basis:           assessment.Basis,
		TargetRole:      assessment.TargetRole,
		TargetGrade:     assessment.TargetGrade,
		AsOfDate:        s.store.Meta().AsOfDate,
		Recommendations: items,
	}
	if includePlan {
		result.LearningPlan = s.learningPlan(assessment, profile, planEvents)
	}
	return result, nil
}

func alignment(employee domain.Employee, event domain.Event) (string, float64, bool) {
	current := contains(event.TargetRoles, employee.Role) && contains(event.TargetGrades, employee.Grade)
	if employee.CareerGoal == nil {
		if current {
			return "current_role", 1, true
		}
		return "", 0, false
	}
	target := contains(event.TargetRoles, employee.CareerGoal.TargetRole) && contains(event.TargetGrades, employee.CareerGoal.TargetGrade)
	if target {
		return "career_goal", 1, true
	}
	if current {
		return "current_role", .5, true
	}
	return "", 0, false
}

func meetsPrerequisites(levels map[string]int, prerequisites map[string]int) bool {
	for skillID, required := range prerequisites {
		if levels[skillID] < required {
			return false
		}
	}
	return true
}

func availability(event domain.Event, asOfDate string) ([]string, bool) {
	if event.Format == "self_paced" {
		return []string{}, true
	}
	result := make([]string, 0, len(event.UpcomingSessions))
	for _, session := range event.UpcomingSessions {
		if session >= asOfDate {
			result = append(result, session)
		}
	}
	return result, len(result) > 0
}

func explain(item Item, assessment domain.Assessment) string {
	changes := make([]string, 0, len(item.SkillsCovered))
	for _, impact := range item.SkillsCovered {
		label := ""
		if impact.Critical {
			label = ", critical"
		}
		changes = append(changes, fmt.Sprintf("%s from %d to %d (required: %d%s)", impact.SkillName, impact.BeforeLevel, impact.ProjectedLevel, impact.RequiredLevel, label))
	}
	basis := "current role"
	if assessment.Basis == "career_goal" {
		basis = "career goal"
	}
	message := fmt.Sprintf("For your %s %s %s, this activity is projected to improve %s.", assessment.TargetRole, assessment.TargetGrade, basis, strings.Join(changes, "; "))
	if item.ReadinessImpact != nil {
		message += fmt.Sprintf(" Projected career readiness increases by %.1f percentage points.", *item.ReadinessImpact)
	}
	if item.PreviousStatus != "" {
		message += fmt.Sprintf(" A previous attempt was marked %s, so it is ranked slightly lower.", item.PreviousStatus)
	}
	return message
}

func isUnsuccessful(status string) bool {
	return status == "dropped" || status == "no_show" || status == "declined" || status == "failed"
}

func skillWeight(critical bool) float64 {
	if critical {
		return 2
	}
	return 1
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func pointerValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func round1(value float64) float64 { return math.Round(value*10) / 10 }
