package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"careerquest/internal/auth"
	"careerquest/internal/domain"
	"careerquest/internal/repository"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Store struct {
	db   *sql.DB
	meta domain.Meta
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(12)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	return &Store{db: db, meta: domain.Meta{Dataset: "Career Quest", Version: "1.0"}}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
		return err
	}
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		var applied bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)`, entry.Name()).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(name) VALUES($1)`, entry.Name()); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	s.refreshMeta(ctx)
	return nil
}

func (s *Store) refreshMeta(ctx context.Context) {
	var asOf string
	if err := s.db.QueryRowContext(ctx, `SELECT value FROM app_metadata WHERE key='as_of_date'`).Scan(&asOf); err == nil {
		s.meta.AsOfDate = asOf
	}
}

func (s *Store) Meta() domain.Meta { return s.meta }

func (s *Store) ProficiencyScale() map[string]string {
	return map[string]string{
		"0": "No knowledge",
		"1": "Basic awareness: knows key concepts, needs guidance",
		"2": "Working knowledge: handles routine tasks with some support",
		"3": "Proficient: works independently on typical tasks",
		"4": "Advanced: handles complex cases, guides others",
		"5": "Expert: sets standards and shapes practice across the company",
	}
}

func (s *Store) Skill(id string) (domain.Skill, bool) {
	var skill domain.Skill
	err := s.db.QueryRow(`SELECT id,name,type,category,description FROM skills WHERE id=$1`, id).Scan(&skill.ID, &skill.Name, &skill.Type, &skill.Category, &skill.Description)
	return skill, err == nil
}

func (s *Store) Skills() []domain.Skill {
	rows, err := s.db.Query(`SELECT id,name,type,category,description FROM skills ORDER BY name`)
	if err != nil {
		return []domain.Skill{}
	}
	defer rows.Close()
	result := []domain.Skill{}
	for rows.Next() {
		var skill domain.Skill
		if rows.Scan(&skill.ID, &skill.Name, &skill.Type, &skill.Category, &skill.Description) == nil {
			result = append(result, skill)
		}
	}
	return result
}

func (s *Store) Profile(role, grade string) (domain.RoleProfile, bool) {
	profile := domain.RoleProfile{Role: role, Grade: grade, RequiredSkills: map[string]int{}, CriticalSkills: []string{}}
	rows, err := s.db.Query(`
		SELECT rsr.skill_id,rsr.required_level,rsr.critical
		FROM role_skill_requirements rsr
		JOIN job_roles jr ON jr.id=rsr.job_role_id JOIN grades g ON g.id=rsr.grade_id
		WHERE jr.name=$1 AND g.name=$2 ORDER BY rsr.skill_id`, role, grade)
	if err != nil {
		return domain.RoleProfile{}, false
	}
	defer rows.Close()
	for rows.Next() {
		var skillID string
		var level int
		var critical bool
		if rows.Scan(&skillID, &level, &critical) != nil {
			return domain.RoleProfile{}, false
		}
		profile.RequiredSkills[skillID] = level
		if critical {
			profile.CriticalSkills = append(profile.CriticalSkills, skillID)
		}
	}
	return profile, len(profile.RequiredSkills) > 0
}

func (s *Store) Profiles() []domain.RoleProfile {
	rows, err := s.db.Query(`SELECT jr.name,g.name FROM role_skill_requirements r JOIN job_roles jr ON jr.id=r.job_role_id JOIN grades g ON g.id=r.grade_id GROUP BY jr.name,g.name,g.rank ORDER BY jr.name,g.rank`)
	if err != nil {
		return []domain.RoleProfile{}
	}
	defer rows.Close()
	result := []domain.RoleProfile{}
	for rows.Next() {
		var role, grade string
		if rows.Scan(&role, &grade) == nil {
			if profile, ok := s.Profile(role, grade); ok {
				result = append(result, profile)
			}
		}
	}
	return result
}

const employeeSelect = `
	SELECT e.id,e.full_name,COALESCE(e.email,''),COALESCE(e.phone,''),e.department,e.team,
	       e.manager_id,COALESCE(m.full_name,''),e.location,jr.name,g.name,e.hire_date,e.tenure_months,
	       e.work_format,e.preferred_language,e.last_review_date,
	       COALESCE(cjr.name,''),COALESCE(cg.name,'')
	FROM employees e
	JOIN job_roles jr ON jr.id=e.job_role_id JOIN grades g ON g.id=e.grade_id
	LEFT JOIN employees m ON m.id=e.manager_id
	LEFT JOIN career_goals goal ON goal.employee_id=e.id
	LEFT JOIN job_roles cjr ON cjr.id=goal.target_job_role_id
	LEFT JOIN grades cg ON cg.id=goal.target_grade_id`

func scanEmployee(scanner interface{ Scan(...any) error }) (domain.Employee, error) {
	var employee domain.Employee
	var managerID sql.NullString
	var managerName, hireDate, reviewDate, goalRole, goalGrade string
	var hire, review sql.NullTime
	err := scanner.Scan(&employee.ID, &employee.FullName, &employee.Email, &employee.Phone, &employee.Department, &employee.Team,
		&managerID, &managerName, &employee.Location, &employee.Role, &employee.Grade, &hire, &employee.TenureMonths,
		&employee.WorkFormat, &employee.PreferredLanguage, &review, &goalRole, &goalGrade)
	if err != nil {
		return employee, err
	}
	if managerID.Valid {
		employee.ManagerID = &managerID.String
		employee.ManagerName = managerName
	}
	if hire.Valid {
		hireDate = hire.Time.Format("2006-01-02")
	}
	if review.Valid {
		reviewDate = review.Time.Format("2006-01-02")
	}
	employee.HireDate, employee.LastReviewDate = hireDate, reviewDate
	if goalRole != "" {
		employee.CareerGoal = &domain.CareerGoal{TargetRole: goalRole, TargetGrade: goalGrade}
	}
	employee.Skills = map[string]int{}
	return employee, nil
}

func (s *Store) Employee(id string) (domain.Employee, bool) {
	employee, err := scanEmployee(s.db.QueryRow(employeeSelect+` WHERE e.id=$1`, id))
	if err != nil {
		return domain.Employee{}, false
	}
	employee.Skills = s.employeeSkills(id)
	return employee, true
}

func (s *Store) employeeSkills(employeeID string) map[string]int {
	result := map[string]int{}
	rows, err := s.db.Query(`SELECT skill_id,level FROM employee_skills WHERE employee_id=$1`, employeeID)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var level int
		if rows.Scan(&id, &level) == nil {
			result[id] = level
		}
	}
	return result
}

func (s *Store) Employees() []domain.Employee {
	employees, _ := s.SearchEmployees(repository.EmployeeSearch{})
	return employees
}

func (s *Store) SearchEmployees(filter repository.EmployeeSearch) ([]domain.Employee, error) {
	rows, err := s.db.Query(employeeSelect+`
		WHERE ($1='' OR CONCAT_WS(' ',e.id,e.full_name,e.email,e.department,e.team,jr.name,g.name) ILIKE '%'||$1||'%')
		  AND ($2='' OR e.department=$2) AND ($3='' OR e.team=$3)
		  AND ($4='' OR jr.name=$4) AND ($5='' OR g.name=$5)
		ORDER BY e.full_name,e.id`, filter.Query, filter.Department, filter.Team, filter.Role, filter.Grade)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.Employee{}
	for rows.Next() {
		employee, err := scanEmployee(rows)
		if err != nil {
			return nil, err
		}
		employee.Skills = s.employeeSkills(employee.ID)
		result = append(result, employee)
	}
	return result, rows.Err()
}

func (s *Store) UpdateCareerGoal(id string, goal *domain.CareerGoal) (domain.Employee, error) {
	if goal == nil {
		if _, err := s.db.Exec(`DELETE FROM career_goals WHERE employee_id=$1`, id); err != nil {
			return domain.Employee{}, err
		}
	} else {
		result, err := s.db.Exec(`
			INSERT INTO career_goals(employee_id,target_job_role_id,target_grade_id)
			SELECT $1,jr.id,g.id FROM job_roles jr CROSS JOIN grades g WHERE jr.name=$2 AND g.name=$3
			ON CONFLICT(employee_id) DO UPDATE SET target_job_role_id=EXCLUDED.target_job_role_id,target_grade_id=EXCLUDED.target_grade_id,updated_at=NOW()`, id, goal.TargetRole, goal.TargetGrade)
		if err != nil {
			return domain.Employee{}, err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return domain.Employee{}, errors.New("unknown target role or grade")
		}
	}
	employee, ok := s.Employee(id)
	if !ok {
		return domain.Employee{}, fmt.Errorf("employee %q not found", id)
	}
	return employee, nil
}

func (s *Store) Event(id string) (domain.Event, bool) {
	event := domain.Event{DevelopsSkills: []domain.SkillEffect{}}
	err := s.db.QueryRow(`SELECT id,title,description,type,format,duration_hours,mandatory,learning_link FROM events WHERE id=$1`, id).Scan(
		&event.ID, &event.Title, &event.Description, &event.Type, &event.Format, &event.DurationHours, &event.Mandatory, &event.LearningLink)
	if err != nil {
		return domain.Event{}, false
	}
	event.TargetRoles = queryStrings(s.db, `SELECT jr.name FROM event_target_roles t JOIN job_roles jr ON jr.id=t.job_role_id WHERE t.event_id=$1 ORDER BY jr.name`, id)
	event.TargetGrades = queryStrings(s.db, `SELECT g.name FROM event_target_grades t JOIN grades g ON g.id=t.grade_id WHERE t.event_id=$1 ORDER BY g.rank`, id)
	event.UpcomingSessions = queryStrings(s.db, `SELECT session_date::text FROM course_sessions WHERE event_id=$1 ORDER BY session_date`, id)
	event.Prerequisites = map[string]int{}
	rows, _ := s.db.Query(`SELECT skill_id,minimum_level FROM event_prerequisites WHERE event_id=$1`, id)
	if rows != nil {
		for rows.Next() {
			var skill string
			var level int
			if rows.Scan(&skill, &level) == nil {
				event.Prerequisites[skill] = level
			}
		}
		rows.Close()
	}
	effects, _ := s.db.Query(`SELECT skill_id,gain,max_level FROM event_skill_effects WHERE event_id=$1 ORDER BY skill_id`, id)
	if effects != nil {
		for effects.Next() {
			var effect domain.SkillEffect
			if effects.Scan(&effect.SkillID, &effect.Gain, &effect.MaxLevel) == nil {
				event.DevelopsSkills = append(event.DevelopsSkills, effect)
			}
		}
		effects.Close()
	}
	return event, true
}

func queryStrings(db *sql.DB, query string, args ...any) []string {
	rows, err := db.Query(query, args...)
	if err != nil {
		return []string{}
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var value string
		if rows.Scan(&value) == nil {
			result = append(result, value)
		}
	}
	return result
}

func (s *Store) Events() []domain.Event {
	events, _ := s.SearchEvents(repository.EventSearch{})
	return events
}

func (s *Store) SearchEvents(filter repository.EventSearch) ([]domain.Event, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT e.id
		FROM events e
		LEFT JOIN event_target_roles etr ON etr.event_id=e.id LEFT JOIN job_roles jr ON jr.id=etr.job_role_id
		LEFT JOIN event_target_grades etg ON etg.event_id=e.id LEFT JOIN grades g ON g.id=etg.grade_id
		LEFT JOIN event_skill_effects ese ON ese.event_id=e.id LEFT JOIN skills s ON s.id=ese.skill_id
		WHERE ($1='' OR CONCAT_WS(' ',e.id,e.title,e.description,e.type,e.format,jr.name,g.name,s.name) ILIKE '%'||$1||'%')
		  AND ($2='' OR e.type=$2)
		  AND ($3='' OR EXISTS(SELECT 1 FROM event_target_roles x JOIN job_roles r ON r.id=x.job_role_id WHERE x.event_id=e.id AND r.name=$3))
		  AND ($4='' OR EXISTS(SELECT 1 FROM event_target_grades x JOIN grades gr ON gr.id=x.grade_id WHERE x.event_id=e.id AND gr.name=$4))
		ORDER BY e.id`, filter.Query, filter.Type, filter.Role, filter.Grade)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	result := make([]domain.Event, 0, len(ids))
	for _, id := range ids {
		if event, ok := s.Event(id); ok {
			result = append(result, event)
		}
	}
	return result, rows.Err()
}

func randomID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + strings.ToUpper(hex.EncodeToString(b))
}

func (s *Store) CreateEvent(event domain.Event) (domain.Event, error) {
	event.ID = randomID("EV_")
	return s.saveEvent(event, false)
}

func (s *Store) UpdateEvent(id string, event domain.Event) (domain.Event, error) {
	event.ID = id
	return s.saveEvent(event, true)
}

func (s *Store) saveEvent(event domain.Event, update bool) (domain.Event, error) {
	if strings.TrimSpace(event.Title) == "" || event.DurationHours <= 0 || len(event.TargetRoles) == 0 || len(event.TargetGrades) == 0 {
		return domain.Event{}, errors.New("title, positive duration, target roles and target grades are required")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Event{}, err
	}
	defer tx.Rollback()
	if update {
		result, err := tx.Exec(`UPDATE events SET title=$2,description=$3,type=$4,format=$5,duration_hours=$6,mandatory=$7,learning_link=$8,updated_at=NOW() WHERE id=$1`, event.ID, event.Title, event.Description, event.Type, event.Format, event.DurationHours, event.Mandatory, event.LearningLink)
		if err != nil {
			return domain.Event{}, err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return domain.Event{}, fmt.Errorf("event %q not found", event.ID)
		}
		for _, table := range []string{"event_target_roles", "event_target_grades", "event_skill_effects", "event_prerequisites", "course_sessions"} {
			if _, err := tx.Exec(`DELETE FROM `+table+` WHERE event_id=$1`, event.ID); err != nil {
				return domain.Event{}, err
			}
		}
	} else if _, err := tx.Exec(`INSERT INTO events(id,title,description,type,format,duration_hours,mandatory,learning_link) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, event.ID, event.Title, event.Description, event.Type, event.Format, event.DurationHours, event.Mandatory, event.LearningLink); err != nil {
		return domain.Event{}, err
	}
	if err := insertEventRelations(tx, event); err != nil {
		return domain.Event{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Event{}, err
	}
	saved, ok := s.Event(event.ID)
	if !ok {
		return domain.Event{}, errors.New("saved event could not be loaded")
	}
	return saved, nil
}

func insertEventRelations(tx *sql.Tx, event domain.Event) error {
	for _, role := range event.TargetRoles {
		if result, err := tx.Exec(`INSERT INTO event_target_roles(event_id,job_role_id) SELECT $1,id FROM job_roles WHERE name=$2`, event.ID, role); err != nil {
			return err
		} else if count, _ := result.RowsAffected(); count == 0 {
			return fmt.Errorf("unknown target role %q", role)
		}
	}
	for _, grade := range event.TargetGrades {
		if result, err := tx.Exec(`INSERT INTO event_target_grades(event_id,grade_id) SELECT $1,id FROM grades WHERE name=$2`, event.ID, grade); err != nil {
			return err
		} else if count, _ := result.RowsAffected(); count == 0 {
			return fmt.Errorf("unknown target grade %q", grade)
		}
	}
	for _, effect := range event.DevelopsSkills {
		if _, err := tx.Exec(`INSERT INTO event_skill_effects(event_id,skill_id,gain,max_level) VALUES($1,$2,$3,$4)`, event.ID, effect.SkillID, effect.Gain, effect.MaxLevel); err != nil {
			return err
		}
	}
	for skill, level := range event.Prerequisites {
		if _, err := tx.Exec(`INSERT INTO event_prerequisites(event_id,skill_id,minimum_level) VALUES($1,$2,$3)`, event.ID, skill, level); err != nil {
			return err
		}
	}
	for _, date := range event.UpcomingSessions {
		if _, err := tx.Exec(`INSERT INTO course_sessions(event_id,session_date) VALUES($1,$2)`, event.ID, date); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ActivitiesForEmployee(id string) []domain.Activity {
	rows, err := s.db.Query(`SELECT COALESCE(enrollment_id,0),id,employee_id,event_id,activity_date::text,COALESCE(due_date::text,''),status,completion_pct,score,feedback_rating,assigned_by,COALESCE((SELECT skill_rewards_applied FROM assessments WHERE assessments.enrollment_id=activity_history.enrollment_id),FALSE) FROM activity_history WHERE employee_id=$1 ORDER BY activity_date,id`, id)
	if err != nil {
		return []domain.Activity{}
	}
	defer rows.Close()
	result := []domain.Activity{}
	for rows.Next() {
		var activity domain.Activity
		var score, feedback sql.NullInt64
		if rows.Scan(&activity.EnrollmentID, &activity.RecordID, &activity.EmployeeID, &activity.EventID, &activity.Date, &activity.DueDate, &activity.Status, &activity.CompletionPct, &score, &feedback, &activity.AssignedBy, &activity.SkillRewardsApplied) == nil {
			if score.Valid {
				value := int(score.Int64)
				activity.Score = &value
			}
			if feedback.Valid {
				value := int(feedback.Int64)
				activity.Feedback = &value
			}
			result = append(result, activity)
		}
	}
	return result
}

func (s *Store) CreateEmployee(input repository.EmployeeCreate) (domain.Employee, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Employee{}, err
	}
	defer tx.Rollback()
	id, err := insertEmployee(tx, input)
	if err != nil {
		return domain.Employee{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Employee{}, err
	}
	employee, ok := s.Employee(id)
	if !ok {
		return domain.Employee{}, errors.New("created employee could not be loaded")
	}
	return employee, nil
}

func insertEmployee(tx *sql.Tx, input repository.EmployeeCreate) (string, error) {
	input.FullName = strings.TrimSpace(input.FullName)
	input.Role = strings.TrimSpace(input.Role)
	input.Grade = strings.TrimSpace(input.Grade)
	if input.FullName == "" || input.Role == "" || input.Grade == "" {
		return "", errors.New("full_name, role and grade are required")
	}
	if input.ID == "" {
		var number int64
		if err := tx.QueryRow(`SELECT nextval('employee_id_seq')`).Scan(&number); err != nil {
			return "", err
		}
		input.ID = fmt.Sprintf("E%04d", number)
	}
	var hire any
	if input.HireDate != "" {
		hire = input.HireDate
	}
	result, err := tx.Exec(`INSERT INTO employees(id,full_name,email,phone,department,team,manager_id,location,job_role_id,grade_id,hire_date,work_format,preferred_language)
		SELECT $1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,$7,$8,jr.id,g.id,$9,$10,$11 FROM job_roles jr CROSS JOIN grades g WHERE jr.name=$12 AND g.name=$13`, input.ID, input.FullName, input.Email, input.Phone, input.Department, input.Team, input.ManagerID, input.Location, hire, input.WorkFormat, input.PreferredLanguage, input.Role, input.Grade)
	if err != nil {
		return "", employeeWriteError(err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return "", errors.New("unknown role or grade")
	}
	return input.ID, nil
}

func employeeWriteError(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		switch databaseError.ConstraintName {
		case "employees_email_unique":
			return errors.New("an employee with this email already exists; link the existing employee instead")
		case "employees_pkey":
			return errors.New("employee ID already exists")
		}
	}
	return err
}

func (s *Store) FindAccountByEmail(email string) (auth.User, string, error) {
	var user auth.User
	var roleCode, password string
	var employeeID sql.NullString
	err := s.db.QueryRow(`SELECT u.id,u.email,u.name,u.password_hash,u.status,r.code,u.employee_id FROM users u JOIN application_roles r ON r.id=u.application_role_id WHERE LOWER(u.email)=LOWER($1)`, email).Scan(&user.ID, &user.Email, &user.Name, &password, &user.Status, &roleCode, &employeeID)
	if err != nil {
		return auth.User{}, "", err
	}
	user.Role = roleFromDB(roleCode)
	if employeeID.Valid {
		user.EmployeeID = employeeID.String
	}
	return user, password, nil
}

func roleFromDB(code string) auth.Role {
	switch code {
	case "HR":
		return auth.RoleHR
	case "LD_SPECIALIST":
		return auth.RoleLD
	default:
		return auth.RoleEmployee
	}
}

func (s *Store) CreatePendingAccount(input auth.RegistrationInput, passwordHash string) (auth.User, error) {
	id := randomID("U_")
	var user auth.User
	err := s.db.QueryRow(`INSERT INTO users(id,email,name,password_hash,status,application_role_id,requested_employee_id) SELECT $1,LOWER($2),$3,$4,'PENDING',id,$5 FROM application_roles WHERE code='EMPLOYEE' RETURNING id,email,name,status,requested_employee_id`, id, input.Email, input.Name, passwordHash, input.EmployeeID).Scan(&user.ID, &user.Email, &user.Name, &user.Status, &user.RequestedEmployeeID)
	if err != nil {
		return auth.User{}, accountWriteError(err)
	}
	user.Role = auth.RoleEmployee
	return user, nil
}

func (s *Store) ListRegistrations(status string) ([]auth.User, error) {
	rows, err := s.db.Query(`SELECT u.id,u.email,u.name,u.status,r.code,COALESCE(u.employee_id,''),u.requested_employee_id FROM users u JOIN application_roles r ON r.id=u.application_role_id WHERE ($1='' OR u.status=$1) ORDER BY u.created_at`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []auth.User{}
	for rows.Next() {
		var user auth.User
		var role string
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.Status, &role, &user.EmployeeID, &user.RequestedEmployeeID); err != nil {
			return nil, err
		}
		user.Role = roleFromDB(role)
		result = append(result, user)
	}
	return result, rows.Err()
}

func (s *Store) UpdateRegistration(userID, status, employeeID string) (auth.User, error) {
	if status == "ACTIVE" && employeeID == "" {
		return auth.User{}, errors.New("employee_id is required when approving a registration")
	}
	var employee any
	if employeeID != "" {
		employee = employeeID
	}
	result, err := s.db.Exec(`UPDATE users SET status=$2,employee_id=CASE WHEN $2='ACTIVE' THEN $3 ELSE employee_id END,updated_at=NOW() WHERE id=$1 AND status='PENDING'`, userID, status, employee)
	if err != nil {
		return auth.User{}, accountWriteError(err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return auth.User{}, errors.New("pending registration not found")
	}
	var email string
	if err := s.db.QueryRow(`SELECT email FROM users WHERE id=$1`, userID).Scan(&email); err != nil {
		return auth.User{}, err
	}
	user, _, err := s.FindAccountByEmail(email)
	return user, err
}

func (s *Store) ApproveWithNewEmployee(userID string, input auth.NewEmployeeApproval) (auth.User, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return auth.User{}, err
	}
	defer tx.Rollback()
	var name, email, status string
	if err := tx.QueryRow(`SELECT name,email,status FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&name, &email, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.User{}, errors.New("pending registration not found")
		}
		return auth.User{}, err
	}
	if status != "PENDING" {
		return auth.User{}, errors.New("pending registration not found")
	}
	id, err := insertEmployee(tx, repository.EmployeeCreate{
		FullName: name, Email: email, Department: input.Department, Team: input.Team,
		Role: input.Role, Grade: input.Grade,
	})
	if err != nil {
		return auth.User{}, err
	}
	if _, err := tx.Exec(`UPDATE users SET status='ACTIVE',employee_id=$2,updated_at=NOW() WHERE id=$1`, userID, id); err != nil {
		return auth.User{}, accountWriteError(err)
	}
	if err := tx.Commit(); err != nil {
		return auth.User{}, err
	}
	user, _, err := s.FindAccountByEmail(email)
	return user, err
}

func accountWriteError(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		switch databaseError.ConstraintName {
		case "users_email_unique":
			return errors.New("an account with this email already exists; sign in or contact HR")
		case "users_employee_id_fkey":
			return errors.New("employee record not found; create the employee or select an existing employee before approval")
		case "users_employee_id_key":
			return errors.New("this employee is already linked to another account")
		}
	}
	return errors.New("could not save the account; please try again")
}

func (s *Store) EventParticipants(eventID string) ([]repository.Participant, error) {
	rows, err := s.db.Query(`SELECT en.id,e.id,e.full_name,en.status,en.completion_pct,COALESCE(cs.session_date::text,''),COALESCE(a.result,''),a.score,COALESCE(a.feedback,'') FROM enrollments en JOIN employees e ON e.id=en.employee_id LEFT JOIN course_sessions cs ON cs.id=en.session_id LEFT JOIN assessments a ON a.enrollment_id=en.id WHERE en.event_id=$1 ORDER BY e.full_name,en.id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []repository.Participant{}
	for rows.Next() {
		var p repository.Participant
		var score sql.NullInt64
		if err := rows.Scan(&p.EnrollmentID, &p.EmployeeID, &p.FullName, &p.Status, &p.CompletionPct, &p.SessionDate, &p.Result, &score, &p.Feedback); err != nil {
			return nil, err
		}
		if score.Valid {
			v := int(score.Int64)
			p.Score = &v
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *Store) AssessEnrollment(enrollmentID int64, assessorUserID string, input repository.AssessmentInput) (repository.AssessmentResult, error) {
	if input.Result != "PASSED" && input.Result != "FAILED" {
		return repository.AssessmentResult{}, errors.New("result must be PASSED or FAILED")
	}
	if input.Score != nil && (*input.Score < 0 || *input.Score > 100) {
		return repository.AssessmentResult{}, errors.New("score must be between 0 and 100")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return repository.AssessmentResult{}, err
	}
	defer tx.Rollback()
	var employeeID, eventID string
	if err := tx.QueryRow(`SELECT employee_id,event_id FROM enrollments WHERE id=$1 FOR UPDATE`, enrollmentID).Scan(&employeeID, &eventID); err != nil {
		return repository.AssessmentResult{}, errors.New("enrollment not found")
	}
	var alreadyApplied bool
	err = tx.QueryRow(`SELECT skill_rewards_applied FROM assessments WHERE enrollment_id=$1 FOR UPDATE`, enrollmentID).Scan(&alreadyApplied)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return repository.AssessmentResult{}, err
	}
	if alreadyApplied {
		return repository.AssessmentResult{}, errors.New("skill rewards were already applied for this enrollment")
	}
	applied := false
	if input.Result == "PASSED" {
		rows, err := tx.Query(`SELECT skill_id,gain,max_level FROM event_skill_effects WHERE event_id=$1`, eventID)
		if err != nil {
			return repository.AssessmentResult{}, err
		}
		type effect struct {
			id        string
			gain, max int
		}
		effects := []effect{}
		for rows.Next() {
			var e effect
			if err := rows.Scan(&e.id, &e.gain, &e.max); err != nil {
				rows.Close()
				return repository.AssessmentResult{}, err
			}
			effects = append(effects, e)
		}
		rows.Close()
		for _, e := range effects {
			if _, err := tx.Exec(`INSERT INTO employee_skills(employee_id,skill_id,level) VALUES($1,$2,LEAST($3::smallint,$4::smallint)) ON CONFLICT(employee_id,skill_id) DO UPDATE SET level=GREATEST(employee_skills.level,LEAST($4::smallint,(employee_skills.level+$3::smallint)::smallint)),updated_at=NOW()`, employeeID, e.id, e.gain, e.max); err != nil {
				return repository.AssessmentResult{}, err
			}
		}
		applied = true
	}
	_, err = tx.Exec(`INSERT INTO assessments(enrollment_id,employee_id,event_id,result,score,feedback,assessed_by,skill_rewards_applied) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(enrollment_id) DO UPDATE SET result=EXCLUDED.result,score=EXCLUDED.score,feedback=EXCLUDED.feedback,assessed_by=EXCLUDED.assessed_by,assessed_at=NOW(),skill_rewards_applied=EXCLUDED.skill_rewards_applied`, enrollmentID, employeeID, eventID, input.Result, input.Score, input.Feedback, assessorUserID, applied)
	if err != nil {
		return repository.AssessmentResult{}, err
	}
	status := "failed"
	completion := 100
	if input.Result == "PASSED" {
		status = "completed"
	}
	if _, err := tx.Exec(`UPDATE enrollments SET status=$2,completion_pct=$3,updated_at=NOW() WHERE id=$1`, enrollmentID, status, completion); err != nil {
		return repository.AssessmentResult{}, err
	}
	var score any
	if input.Score != nil {
		score = *input.Score
	}
	activityResult, err := tx.Exec(`UPDATE activity_history SET status=$2,completion_pct=100,score=COALESCE($3,score) WHERE enrollment_id=$1`, enrollmentID, status, score)
	if err != nil {
		return repository.AssessmentResult{}, err
	}
	if changed, _ := activityResult.RowsAffected(); changed == 0 {
		if _, err := tx.Exec(`INSERT INTO activity_history(id,enrollment_id,employee_id,event_id,activity_date,status,completion_pct,score,assigned_by) VALUES($1,$2,$3,$4,CURRENT_DATE,$5,100,$6,'L&D assessment')`, randomID("R_ASSESS_"), enrollmentID, employeeID, eventID, status, score); err != nil {
			return repository.AssessmentResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return repository.AssessmentResult{}, err
	}
	return repository.AssessmentResult{EnrollmentID: enrollmentID, Result: input.Result, Score: input.Score, Feedback: input.Feedback, SkillRewardsApplied: applied}, nil
}

var _ repository.Store = (*Store)(nil)
var _ repository.EmployeeManager = (*Store)(nil)
var _ repository.AssessmentManager = (*Store)(nil)
var _ auth.AccountRepository = (*Store)(nil)
