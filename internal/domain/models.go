package domain

// Meta describes the snapshot represented by a dataset document.
type Meta struct {
	Dataset  string `json:"dataset"`
	Version  string `json:"version"`
	AsOfDate string `json:"as_of_date"`
}

type Skill struct {
	ID          string `json:"skill_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type RoleProfile struct {
	Role           string         `json:"role"`
	Grade          string         `json:"grade"`
	RequiredSkills map[string]int `json:"required_skills"`
	CriticalSkills []string       `json:"critical_skills"`
}

type CareerGoal struct {
	TargetRole  string `json:"target_role"`
	TargetGrade string `json:"target_grade"`
}

type Employee struct {
	ID                string         `json:"employee_id"`
	FullName          string         `json:"full_name"`
	Department        string         `json:"department"`
	Role              string         `json:"role"`
	Grade             string         `json:"grade"`
	ManagerID         *string        `json:"manager_id"`
	HireDate          string         `json:"hire_date"`
	TenureMonths      int            `json:"tenure_months"`
	WorkFormat        string         `json:"work_format"`
	PreferredLanguage string         `json:"preferred_language"`
	CareerGoal        *CareerGoal    `json:"career_goal"`
	Skills            map[string]int `json:"skills"`
	LastReviewDate    string         `json:"last_review_date"`
}

type SkillEffect struct {
	SkillID  string `json:"skill_id"`
	Gain     int    `json:"gain"`
	MaxLevel int    `json:"max_level"`
}

type Event struct {
	ID               string         `json:"event_id"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	Type             string         `json:"type"`
	Format           string         `json:"format"`
	DurationHours    float64        `json:"duration_hours"`
	Mandatory        bool           `json:"mandatory"`
	TargetRoles      []string       `json:"target_roles"`
	TargetGrades     []string       `json:"target_grades"`
	DevelopsSkills   []SkillEffect  `json:"develops_skills"`
	Prerequisites    map[string]int `json:"prerequisites"`
	UpcomingSessions []string       `json:"upcoming_sessions"`
	LearningLink     string         `json:"learning_link,omitempty"`
}

type Activity struct {
	RecordID      string `json:"record_id"`
	EmployeeID    string `json:"employee_id"`
	EventID       string `json:"event_id"`
	Date          string `json:"date"`
	DueDate       string `json:"due_date,omitempty"`
	Status        string `json:"status"`
	CompletionPct int    `json:"completion_pct"`
	Score         *int   `json:"score,omitempty"`
	Feedback      *int   `json:"feedback_rating,omitempty"`
	AssignedBy    string `json:"assigned_by"`
}

type SkillGap struct {
	SkillID       string `json:"skill_id"`
	SkillName     string `json:"skill_name"`
	CurrentLevel  int    `json:"current_level"`
	RequiredLevel int    `json:"required_level"`
	MissingLevels int    `json:"missing_levels"`
	Critical      bool   `json:"critical"`
}

type Assessment struct {
	EmployeeID       string         `json:"employee_id"`
	Basis            string         `json:"basis"`
	TargetRole       string         `json:"target_role"`
	TargetGrade      string         `json:"target_grade"`
	EffectiveSkills  map[string]int `json:"effective_skills"`
	Gaps             []SkillGap     `json:"skill_gaps"`
	ReadinessPercent *float64       `json:"readiness_percent"`
}
