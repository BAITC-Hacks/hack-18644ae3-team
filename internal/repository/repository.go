package repository

import "careerquest/internal/domain"

type EmployeeSearch struct {
	Query      string
	Department string
	Team       string
	Role       string
	Grade      string
}

type EventSearch struct {
	Query string
	Type  string
	Role  string
	Grade string
}

// Store is the persistence boundary used by career and recommendation logic.
// The dataset adapter implements it for deterministic unit tests; PostgreSQL is
// the production implementation.
type Store interface {
	Meta() domain.Meta
	ProficiencyScale() map[string]string
	Skill(id string) (domain.Skill, bool)
	Skills() []domain.Skill
	Profile(role, grade string) (domain.RoleProfile, bool)
	Profiles() []domain.RoleProfile
	Employee(id string) (domain.Employee, bool)
	Employees() []domain.Employee
	SearchEmployees(filter EmployeeSearch) ([]domain.Employee, error)
	UpdateCareerGoal(id string, goal *domain.CareerGoal) (domain.Employee, error)
	Event(id string) (domain.Event, bool)
	Events() []domain.Event
	SearchEvents(filter EventSearch) ([]domain.Event, error)
	CreateEvent(event domain.Event) (domain.Event, error)
	UpdateEvent(id string, event domain.Event) (domain.Event, error)
	ActivitiesForEmployee(id string) []domain.Activity
}

type EmployeeCreate struct {
	ID                string  `json:"employee_id"`
	FullName          string  `json:"full_name"`
	Email             string  `json:"email,omitempty"`
	Phone             string  `json:"phone,omitempty"`
	Department        string  `json:"department,omitempty"`
	Team              string  `json:"team,omitempty"`
	ManagerID         *string `json:"manager_id,omitempty"`
	Location          string  `json:"location,omitempty"`
	Role              string  `json:"role"`
	Grade             string  `json:"grade"`
	HireDate          string  `json:"hire_date,omitempty"`
	WorkFormat        string  `json:"work_format,omitempty"`
	PreferredLanguage string  `json:"preferred_language,omitempty"`
}

type EmployeeManager interface {
	CreateEmployee(input EmployeeCreate) (domain.Employee, error)
}

type Participant struct {
	EnrollmentID  int64  `json:"enrollment_id"`
	EmployeeID    string `json:"employee_id"`
	FullName      string `json:"full_name"`
	Status        string `json:"status"`
	CompletionPct int    `json:"completion_pct"`
	SessionDate   string `json:"session_date,omitempty"`
	Result        string `json:"result,omitempty"`
	Score         *int   `json:"score,omitempty"`
	Feedback      string `json:"feedback,omitempty"`
}

type AssessmentInput struct {
	Result   string `json:"result"`
	Score    *int   `json:"score,omitempty"`
	Feedback string `json:"feedback,omitempty"`
}

type AssessmentResult struct {
	EnrollmentID        int64  `json:"enrollment_id"`
	Result              string `json:"result"`
	Score               *int   `json:"score,omitempty"`
	Feedback            string `json:"feedback,omitempty"`
	SkillRewardsApplied bool   `json:"skill_rewards_applied"`
}

type AssessmentManager interface {
	EventParticipants(eventID string) ([]Participant, error)
	AssessEnrollment(enrollmentID int64, assessorUserID string, input AssessmentInput) (AssessmentResult, error)
}
