package dataset

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"careerquest/internal/domain"
	"careerquest/internal/repository"
)

const dateLayout = "2006-01-02"

type skillsDocument struct {
	Meta             domain.Meta          `json:"meta"`
	ProficiencyScale map[string]string    `json:"proficiency_scale"`
	Skills           []domain.Skill       `json:"skills"`
	RoleProfiles     []domain.RoleProfile `json:"role_profiles"`
}

type employeesDocument struct {
	Meta      domain.Meta       `json:"meta"`
	Employees []domain.Employee `json:"employees"`
}

type eventsDocument struct {
	Meta   domain.Meta    `json:"meta"`
	Events []domain.Event `json:"events"`
}

// Store is an in-memory, validated snapshot of the supplied dataset.
// Dataset content is immutable except for career goals, which are explicitly
// updated by the API.
type Store struct {
	mu               sync.RWMutex
	meta             domain.Meta
	proficiencyScale map[string]string
	skills           map[string]domain.Skill
	profiles         map[string]domain.RoleProfile
	employees        map[string]domain.Employee
	events           map[string]domain.Event
	activities       map[string][]domain.Activity
}

func Load(dir string) (*Store, error) {
	var skillDoc skillsDocument
	if err := readJSON(filepath.Join(dir, "skills.json"), &skillDoc); err != nil {
		return nil, err
	}
	var employeeDoc employeesDocument
	if err := readJSON(filepath.Join(dir, "employees.json"), &employeeDoc); err != nil {
		return nil, err
	}
	var eventDoc eventsDocument
	if err := readJSON(filepath.Join(dir, "events.json"), &eventDoc); err != nil {
		return nil, err
	}
	activities, err := readActivities(filepath.Join(dir, "activity_history.csv"))
	if err != nil {
		return nil, err
	}

	s := &Store{
		meta:             skillDoc.Meta,
		proficiencyScale: cloneStringMap(skillDoc.ProficiencyScale),
		skills:           make(map[string]domain.Skill, len(skillDoc.Skills)),
		profiles:         make(map[string]domain.RoleProfile, len(skillDoc.RoleProfiles)),
		employees:        make(map[string]domain.Employee, len(employeeDoc.Employees)),
		events:           make(map[string]domain.Event, len(eventDoc.Events)),
		activities:       make(map[string][]domain.Activity),
	}

	if err := validateMatchingMeta(skillDoc.Meta, employeeDoc.Meta, eventDoc.Meta); err != nil {
		return nil, err
	}
	for _, skill := range skillDoc.Skills {
		if skill.ID == "" {
			return nil, errors.New("skills.json: skill_id cannot be empty")
		}
		if _, exists := s.skills[skill.ID]; exists {
			return nil, fmt.Errorf("skills.json: duplicate skill_id %q", skill.ID)
		}
		s.skills[skill.ID] = skill
	}
	for _, profile := range skillDoc.RoleProfiles {
		key := profileKey(profile.Role, profile.Grade)
		if _, exists := s.profiles[key]; exists {
			return nil, fmt.Errorf("skills.json: duplicate role profile %q/%q", profile.Role, profile.Grade)
		}
		s.profiles[key] = cloneProfile(profile)
	}
	for _, employee := range employeeDoc.Employees {
		if employee.ID == "" {
			return nil, errors.New("employees.json: employee_id cannot be empty")
		}
		if _, exists := s.employees[employee.ID]; exists {
			return nil, fmt.Errorf("employees.json: duplicate employee_id %q", employee.ID)
		}
		s.employees[employee.ID] = cloneEmployee(employee)
	}
	for _, event := range eventDoc.Events {
		if event.ID == "" {
			return nil, errors.New("events.json: event_id cannot be empty")
		}
		if _, exists := s.events[event.ID]; exists {
			return nil, fmt.Errorf("events.json: duplicate event_id %q", event.ID)
		}
		s.events[event.ID] = cloneEvent(event)
	}
	for _, activity := range activities {
		s.activities[activity.EmployeeID] = append(s.activities[activity.EmployeeID], activity)
	}
	for id := range s.activities {
		sort.SliceStable(s.activities[id], func(i, j int) bool {
			if s.activities[id][i].Date == s.activities[id][j].Date {
				return s.activities[id][i].RecordID < s.activities[id][j].RecordID
			}
			return s.activities[id][i].Date < s.activities[id][j].Date
		})
	}
	if err := s.validate(); err != nil {
		return nil, err
	}
	return s, nil
}

func readJSON(path string, dst any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func readActivities(path string) ([]domain.Activity, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read %s header: %w", path, err)
	}
	columns := make(map[string]int, len(header))
	for i, name := range header {
		columns[strings.TrimSpace(name)] = i
	}
	required := []string{"record_id", "employee_id", "event_id", "date", "due_date", "status", "completion_pct", "score", "feedback_rating", "assigned_by"}
	for _, name := range required {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("%s: missing column %q", path, name)
		}
	}

	var result []domain.Activity
	for line := 2; ; line++ {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s row %d: %w", path, line, err)
		}
		value := func(name string) string { return strings.TrimSpace(row[columns[name]]) }
		completion, err := strconv.Atoi(value("completion_pct"))
		if err != nil {
			return nil, fmt.Errorf("%s row %d: invalid completion_pct: %w", path, line, err)
		}
		score, err := optionalInt(value("score"))
		if err != nil {
			return nil, fmt.Errorf("%s row %d: invalid score: %w", path, line, err)
		}
		feedback, err := optionalInt(value("feedback_rating"))
		if err != nil {
			return nil, fmt.Errorf("%s row %d: invalid feedback_rating: %w", path, line, err)
		}
		result = append(result, domain.Activity{
			RecordID:      value("record_id"),
			EmployeeID:    value("employee_id"),
			EventID:       value("event_id"),
			Date:          value("date"),
			DueDate:       value("due_date"),
			Status:        value("status"),
			CompletionPct: completion,
			Score:         score,
			Feedback:      feedback,
			AssignedBy:    value("assigned_by"),
		})
	}
	return result, nil
}

func optionalInt(value string) (*int, error) {
	if value == "" {
		return nil, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func validateMatchingMeta(metas ...domain.Meta) error {
	if len(metas) == 0 || metas[0].AsOfDate == "" {
		return errors.New("dataset metadata must include as_of_date")
	}
	if _, err := time.Parse(dateLayout, metas[0].AsOfDate); err != nil {
		return fmt.Errorf("invalid dataset as_of_date %q: %w", metas[0].AsOfDate, err)
	}
	for _, meta := range metas[1:] {
		if meta.Dataset != metas[0].Dataset || meta.Version != metas[0].Version || meta.AsOfDate != metas[0].AsOfDate {
			return errors.New("dataset JSON documents have inconsistent metadata")
		}
	}
	return nil
}

func (s *Store) validate() error {
	for _, profile := range s.profiles {
		for skillID, level := range profile.RequiredSkills {
			if _, ok := s.skills[skillID]; !ok {
				return fmt.Errorf("role profile %s/%s references unknown skill %q", profile.Role, profile.Grade, skillID)
			}
			if level < 0 || level > 5 {
				return fmt.Errorf("role profile %s/%s has invalid level %d for %s", profile.Role, profile.Grade, level, skillID)
			}
		}
		for _, skillID := range profile.CriticalSkills {
			if _, ok := profile.RequiredSkills[skillID]; !ok {
				return fmt.Errorf("role profile %s/%s marks non-required skill %q as critical", profile.Role, profile.Grade, skillID)
			}
		}
	}
	for _, employee := range s.employees {
		if _, ok := s.profiles[profileKey(employee.Role, employee.Grade)]; !ok {
			return fmt.Errorf("employee %s has unknown role profile %s/%s", employee.ID, employee.Role, employee.Grade)
		}
		if _, err := time.Parse(dateLayout, employee.LastReviewDate); err != nil {
			return fmt.Errorf("employee %s has invalid last_review_date: %w", employee.ID, err)
		}
		if employee.CareerGoal != nil {
			if _, ok := s.profiles[profileKey(employee.CareerGoal.TargetRole, employee.CareerGoal.TargetGrade)]; !ok {
				return fmt.Errorf("employee %s has unknown career goal %s/%s", employee.ID, employee.CareerGoal.TargetRole, employee.CareerGoal.TargetGrade)
			}
		}
		for skillID, level := range employee.Skills {
			if _, ok := s.skills[skillID]; !ok {
				return fmt.Errorf("employee %s references unknown skill %q", employee.ID, skillID)
			}
			if level < 0 || level > 5 {
				return fmt.Errorf("employee %s has invalid level %d for %s", employee.ID, level, skillID)
			}
		}
		if employee.ManagerID != nil {
			manager, ok := s.employees[*employee.ManagerID]
			if !ok {
				return fmt.Errorf("employee %s references unknown manager %q", employee.ID, *employee.ManagerID)
			}
			if manager.Grade != "Lead" || manager.Department != employee.Department {
				return fmt.Errorf("employee %s manager %s must be a Lead in the same department", employee.ID, manager.ID)
			}
		}
	}
	for _, event := range s.events {
		for _, effect := range event.DevelopsSkills {
			if _, ok := s.skills[effect.SkillID]; !ok {
				return fmt.Errorf("event %s develops unknown skill %q", event.ID, effect.SkillID)
			}
			if effect.Gain <= 0 || effect.MaxLevel < 0 || effect.MaxLevel > 5 {
				return fmt.Errorf("event %s has invalid effect for %s", event.ID, effect.SkillID)
			}
		}
		for skillID, level := range event.Prerequisites {
			if _, ok := s.skills[skillID]; !ok {
				return fmt.Errorf("event %s has unknown prerequisite %q", event.ID, skillID)
			}
			if level < 0 || level > 5 {
				return fmt.Errorf("event %s has invalid prerequisite level for %s", event.ID, skillID)
			}
		}
	}
	seenRecords := make(map[string]struct{})
	for _, activities := range s.activities {
		for _, activity := range activities {
			if _, exists := seenRecords[activity.RecordID]; exists {
				return fmt.Errorf("duplicate activity record_id %q", activity.RecordID)
			}
			seenRecords[activity.RecordID] = struct{}{}
			if _, ok := s.employees[activity.EmployeeID]; !ok {
				return fmt.Errorf("activity %s references unknown employee %q", activity.RecordID, activity.EmployeeID)
			}
			if _, ok := s.events[activity.EventID]; !ok {
				return fmt.Errorf("activity %s references unknown event %q", activity.RecordID, activity.EventID)
			}
			if _, err := time.Parse(dateLayout, activity.Date); err != nil {
				return fmt.Errorf("activity %s has invalid date: %w", activity.RecordID, err)
			}
			if activity.CompletionPct < 0 || activity.CompletionPct > 100 {
				return fmt.Errorf("activity %s has invalid completion_pct %d", activity.RecordID, activity.CompletionPct)
			}
		}
	}
	return nil
}

func profileKey(role, grade string) string { return role + "\x00" + grade }

func (s *Store) Meta() domain.Meta { return s.meta }

func (s *Store) ProficiencyScale() map[string]string {
	return cloneStringMap(s.proficiencyScale)
}

func (s *Store) Skill(id string) (domain.Skill, bool) {
	skill, ok := s.skills[id]
	return skill, ok
}

func (s *Store) Skills() []domain.Skill {
	result := make([]domain.Skill, 0, len(s.skills))
	for _, skill := range s.skills {
		result = append(result, skill)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (s *Store) Profiles() []domain.RoleProfile {
	result := make([]domain.RoleProfile, 0, len(s.profiles))
	for _, profile := range s.profiles {
		result = append(result, cloneProfile(profile))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Role == result[j].Role {
			return result[i].Grade < result[j].Grade
		}
		return result[i].Role < result[j].Role
	})
	return result
}

func (s *Store) Profile(role, grade string) (domain.RoleProfile, bool) {
	profile, ok := s.profiles[profileKey(role, grade)]
	return cloneProfile(profile), ok
}

func (s *Store) Employee(id string) (domain.Employee, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	employee, ok := s.employees[id]
	return cloneEmployee(employee), ok
}

func (s *Store) Employees() []domain.Employee {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Employee, 0, len(s.employees))
	for _, employee := range s.employees {
		result = append(result, cloneEmployee(employee))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].FullName == result[j].FullName {
			return result[i].ID < result[j].ID
		}
		return result[i].FullName < result[j].FullName
	})
	return result
}

func (s *Store) SearchEmployees(filter repository.EmployeeSearch) ([]domain.Employee, error) {
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	result := make([]domain.Employee, 0)
	for _, employee := range s.Employees() {
		searchable := strings.ToLower(strings.Join([]string{
			employee.ID, employee.FullName, employee.Email, employee.Department,
			employee.Team, employee.Role, employee.Grade,
		}, " "))
		if query != "" && !strings.Contains(searchable, query) {
			continue
		}
		if filter.Department != "" && employee.Department != filter.Department {
			continue
		}
		if filter.Team != "" && employee.Team != filter.Team {
			continue
		}
		if filter.Role != "" && employee.Role != filter.Role {
			continue
		}
		if filter.Grade != "" && employee.Grade != filter.Grade {
			continue
		}
		result = append(result, employee)
	}
	return result, nil
}

func (s *Store) UpdateCareerGoal(id string, goal *domain.CareerGoal) (domain.Employee, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	employee, ok := s.employees[id]
	if !ok {
		return domain.Employee{}, fmt.Errorf("employee %q not found", id)
	}
	if goal != nil {
		if _, ok := s.profiles[profileKey(goal.TargetRole, goal.TargetGrade)]; !ok {
			return domain.Employee{}, fmt.Errorf("unknown role profile %s/%s", goal.TargetRole, goal.TargetGrade)
		}
		copyGoal := *goal
		employee.CareerGoal = &copyGoal
	} else {
		employee.CareerGoal = nil
	}
	s.employees[id] = employee
	return cloneEmployee(employee), nil
}

func (s *Store) Event(id string) (domain.Event, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, ok := s.events[id]
	return cloneEvent(event), ok
}

func (s *Store) Events() []domain.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Event, 0, len(s.events))
	for _, event := range s.events {
		result = append(result, cloneEvent(event))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (s *Store) SearchEvents(filter repository.EventSearch) ([]domain.Event, error) {
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	result := make([]domain.Event, 0)
	for _, event := range s.Events() {
		searchText := event.ID + " " + event.Title + " " + event.Description + " " + strings.Join(event.TargetRoles, " ") + " " + strings.Join(event.TargetGrades, " ")
		for _, effect := range event.DevelopsSkills {
			if skill, ok := s.Skill(effect.SkillID); ok {
				searchText += " " + skill.Name
			}
		}
		if query != "" && !strings.Contains(strings.ToLower(searchText), query) {
			continue
		}
		if filter.Type != "" && event.Type != filter.Type || filter.Role != "" && !slices.Contains(event.TargetRoles, filter.Role) || filter.Grade != "" && !slices.Contains(event.TargetGrades, filter.Grade) {
			continue
		}
		result = append(result, event)
	}
	return result, nil
}

func (s *Store) CreateEvent(event domain.Event) (domain.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sequence := 1; ; sequence++ {
		candidate := fmt.Sprintf("EV_%03d", sequence)
		if _, exists := s.events[candidate]; !exists {
			event.ID = candidate
			break
		}
	}
	if err := s.validateEvent(event); err != nil {
		return domain.Event{}, err
	}
	s.events[event.ID] = cloneEvent(event)
	return cloneEvent(event), nil
}

func (s *Store) UpdateEvent(id string, event domain.Event) (domain.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.events[id]; !exists {
		return domain.Event{}, fmt.Errorf("event %q not found", id)
	}
	event.ID = id
	if err := s.validateEvent(event); err != nil {
		return domain.Event{}, err
	}
	s.events[id] = cloneEvent(event)
	return cloneEvent(event), nil
}

func (s *Store) validateEvent(event domain.Event) error {
	if strings.TrimSpace(event.Title) == "" {
		return errors.New("event title is required")
	}
	if strings.TrimSpace(event.Description) == "" {
		return errors.New("event description is required")
	}
	if event.DurationHours <= 0 {
		return errors.New("duration_hours must be greater than zero")
	}
	if len(event.TargetRoles) == 0 || len(event.TargetGrades) == 0 {
		return errors.New("at least one target role and grade are required")
	}
	validRoles, validGrades := make(map[string]bool), make(map[string]bool)
	for _, profile := range s.profiles {
		validRoles[profile.Role] = true
		validGrades[profile.Grade] = true
	}
	for _, role := range event.TargetRoles {
		if !validRoles[role] {
			return fmt.Errorf("unknown target role %q", role)
		}
	}
	for _, grade := range event.TargetGrades {
		if !validGrades[grade] {
			return fmt.Errorf("unknown target grade %q", grade)
		}
	}
	for _, effect := range event.DevelopsSkills {
		if _, ok := s.skills[effect.SkillID]; !ok {
			return fmt.Errorf("unknown developed skill %q", effect.SkillID)
		}
		if effect.Gain <= 0 || effect.MaxLevel < 1 || effect.MaxLevel > 5 {
			return fmt.Errorf("invalid skill effect for %s", effect.SkillID)
		}
	}
	for skillID, level := range event.Prerequisites {
		if _, ok := s.skills[skillID]; !ok {
			return fmt.Errorf("unknown prerequisite skill %q", skillID)
		}
		if level < 0 || level > 5 {
			return fmt.Errorf("invalid prerequisite level for %s", skillID)
		}
	}
	for _, session := range event.UpcomingSessions {
		if _, err := time.Parse(dateLayout, session); err != nil {
			return fmt.Errorf("invalid upcoming session date %q", session)
		}
	}
	if event.LearningLink != "" {
		parsed, err := url.ParseRequestURI(event.LearningLink)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return errors.New("learning_link must be a valid http or https URL")
		}
	}
	return nil
}

func (s *Store) ActivitiesForEmployee(id string) []domain.Activity {
	activities := s.activities[id]
	return append([]domain.Activity(nil), activities...)
}

func cloneEmployee(employee domain.Employee) domain.Employee {
	employee.Skills = cloneIntMap(employee.Skills)
	if employee.CareerGoal != nil {
		goal := *employee.CareerGoal
		employee.CareerGoal = &goal
	}
	if employee.ManagerID != nil {
		managerID := *employee.ManagerID
		employee.ManagerID = &managerID
	}
	return employee
}

func cloneEvent(event domain.Event) domain.Event {
	event.TargetRoles = append([]string(nil), event.TargetRoles...)
	event.TargetGrades = append([]string(nil), event.TargetGrades...)
	event.DevelopsSkills = append([]domain.SkillEffect(nil), event.DevelopsSkills...)
	event.Prerequisites = cloneIntMap(event.Prerequisites)
	event.UpcomingSessions = append([]string(nil), event.UpcomingSessions...)
	return event
}

func cloneProfile(profile domain.RoleProfile) domain.RoleProfile {
	profile.RequiredSkills = cloneIntMap(profile.RequiredSkills)
	profile.CriticalSkills = append([]string(nil), profile.CriticalSkills...)
	return profile
}

func cloneIntMap(src map[string]int) map[string]int {
	dst := make(map[string]int, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
