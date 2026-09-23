package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"careerquest/internal/auth"
	"careerquest/internal/repository"
)

// Seed imports the supplied dataset transactionally. Every insert uses a
// natural key and ON CONFLICT, so repeated startup seeding never duplicates
// records and does not overwrite runtime skill progress or event edits.
func (s *Store) Seed(ctx context.Context, source repository.Store) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, code := range []string{"HR", "EMPLOYEE", "LD_SPECIALIST"} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO application_roles(code) VALUES($1) ON CONFLICT(code) DO NOTHING`, code); err != nil {
			return err
		}
	}
	for rank, grade := range []string{"Junior", "Middle", "Senior", "Lead"} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO grades(name,rank) VALUES($1,$2) ON CONFLICT(name) DO NOTHING`, grade, rank+1); err != nil {
			return err
		}
	}
	for _, skill := range source.Skills() {
		if _, err := tx.ExecContext(ctx, `INSERT INTO skills(id,name,type,category,description) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO NOTHING`, skill.ID, skill.Name, skill.Type, skill.Category, skill.Description); err != nil {
			return err
		}
	}
	for _, profile := range source.Profiles() {
		if _, err := tx.ExecContext(ctx, `INSERT INTO job_roles(name) VALUES($1) ON CONFLICT(name) DO NOTHING`, profile.Role); err != nil {
			return err
		}
		critical := make(map[string]bool, len(profile.CriticalSkills))
		for _, skillID := range profile.CriticalSkills {
			critical[skillID] = true
		}
		for skillID, level := range profile.RequiredSkills {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO role_skill_requirements(job_role_id,grade_id,skill_id,required_level,critical)
				SELECT jr.id,g.id,$3,$4,$5 FROM job_roles jr CROSS JOIN grades g WHERE jr.name=$1 AND g.name=$2
				ON CONFLICT(job_role_id,grade_id,skill_id) DO NOTHING`, profile.Role, profile.Grade, skillID, level, critical[skillID]); err != nil {
				return err
			}
		}
	}

	employees := source.Employees()
	for _, employee := range employees {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO employees(id,full_name,email,phone,department,team,location,job_role_id,grade_id,hire_date,tenure_months,work_format,preferred_language,last_review_date)
			SELECT $1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,$7,jr.id,g.id,NULLIF($10,'')::date,$11,$12,$13,NULLIF($14,'')::date
			FROM job_roles jr CROSS JOIN grades g WHERE jr.name=$8 AND g.name=$9
			ON CONFLICT(id) DO NOTHING`, employee.ID, employee.FullName, employee.Email, employee.Phone, employee.Department, employee.Team, employee.Location,
			employee.Role, employee.Grade, employee.HireDate, employee.TenureMonths, employee.WorkFormat, employee.PreferredLanguage, employee.LastReviewDate); err != nil {
			return fmt.Errorf("seed employee %s: %w", employee.ID, err)
		}
	}
	for _, employee := range employees {
		if employee.ManagerID != nil {
			if _, err := tx.ExecContext(ctx, `UPDATE employees SET manager_id=COALESCE(manager_id,$2) WHERE id=$1`, employee.ID, *employee.ManagerID); err != nil {
				return err
			}
		}
		for skillID, level := range employee.Skills {
			if _, err := tx.ExecContext(ctx, `INSERT INTO employee_skills(employee_id,skill_id,level) VALUES($1,$2,$3) ON CONFLICT(employee_id,skill_id) DO NOTHING`, employee.ID, skillID, level); err != nil {
				return err
			}
		}
		if employee.CareerGoal != nil {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO career_goals(employee_id,target_job_role_id,target_grade_id)
				SELECT $1,jr.id,g.id FROM job_roles jr CROSS JOIN grades g WHERE jr.name=$2 AND g.name=$3
				ON CONFLICT(employee_id) DO NOTHING`, employee.ID, employee.CareerGoal.TargetRole, employee.CareerGoal.TargetGrade); err != nil {
				return err
			}
		}
	}

	for _, event := range source.Events() {
		if _, err := tx.ExecContext(ctx, `INSERT INTO events(id,title,description,type,format,duration_hours,mandatory,learning_link) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO NOTHING`, event.ID, event.Title, event.Description, event.Type, event.Format, event.DurationHours, event.Mandatory, event.LearningLink); err != nil {
			return err
		}
		for _, role := range event.TargetRoles {
			if _, err := tx.ExecContext(ctx, `INSERT INTO event_target_roles(event_id,job_role_id) SELECT $1,id FROM job_roles WHERE name=$2 ON CONFLICT DO NOTHING`, event.ID, role); err != nil {
				return err
			}
		}
		for _, grade := range event.TargetGrades {
			if _, err := tx.ExecContext(ctx, `INSERT INTO event_target_grades(event_id,grade_id) SELECT $1,id FROM grades WHERE name=$2 ON CONFLICT DO NOTHING`, event.ID, grade); err != nil {
				return err
			}
		}
		for _, effect := range event.DevelopsSkills {
			if _, err := tx.ExecContext(ctx, `INSERT INTO event_skill_effects(event_id,skill_id,gain,max_level) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, event.ID, effect.SkillID, effect.Gain, effect.MaxLevel); err != nil {
				return err
			}
		}
		for skillID, level := range event.Prerequisites {
			if _, err := tx.ExecContext(ctx, `INSERT INTO event_prerequisites(event_id,skill_id,minimum_level) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, event.ID, skillID, level); err != nil {
				return err
			}
		}
		for _, date := range event.UpcomingSessions {
			if _, err := tx.ExecContext(ctx, `INSERT INTO course_sessions(event_id,session_date) VALUES($1,$2) ON CONFLICT(event_id,session_date) DO NOTHING`, event.ID, date); err != nil {
				return err
			}
		}
	}

	for _, employee := range employees {
		for _, activity := range source.ActivitiesForEmployee(employee.ID) {
			var enrollmentID int64
			err := tx.QueryRowContext(ctx, `
				INSERT INTO enrollments(employee_id,event_id,status,completion_pct,assigned_by,due_date,source_record_id,created_at)
				VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::date,$7,$8::date)
				ON CONFLICT(source_record_id) DO UPDATE SET source_record_id=EXCLUDED.source_record_id
				RETURNING id`, activity.EmployeeID, activity.EventID, activity.Status, activity.CompletionPct, activity.AssignedBy, activity.DueDate, activity.RecordID, activity.Date).Scan(&enrollmentID)
			if err != nil {
				return fmt.Errorf("seed enrollment %s: %w", activity.RecordID, err)
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO activity_history(id,enrollment_id,employee_id,event_id,activity_date,due_date,status,completion_pct,score,feedback_rating,assigned_by)
				VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::date,$7,$8,$9,$10,$11) ON CONFLICT(id) DO NOTHING`,
				activity.RecordID, enrollmentID, activity.EmployeeID, activity.EventID, activity.Date, activity.DueDate, activity.Status, activity.CompletionPct, activity.Score, activity.Feedback, activity.AssignedBy); err != nil {
				return err
			}
			if activity.Score != nil {
				if _, err := tx.ExecContext(ctx, `INSERT INTO assessments(enrollment_id,employee_id,event_id,score) VALUES($1,$2,$3,$4) ON CONFLICT(enrollment_id) DO NOTHING`, enrollmentID, activity.EmployeeID, activity.EventID, activity.Score); err != nil {
					return err
				}
			}
		}
	}

	meta := source.Meta()
	for key, value := range map[string]string{"dataset": meta.Dataset, "version": meta.Version, "as_of_date": meta.AsOfDate} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO app_metadata(key,value) VALUES($1,$2) ON CONFLICT(key) DO NOTHING`, key, value); err != nil {
			return err
		}
	}
	if err := seedDemoUsers(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.meta = meta
	return nil
}

func seedDemoUsers(ctx context.Context, tx *sql.Tx) error {
	accounts := []struct {
		id, email, name, role, employeeID string
	}{
		{"U_HR_001", "hr@careerquest.demo", "Aigerim Sarsenova", "HR", ""},
		{"U_EMP_001", "employee@careerquest.demo", "Marat Yessenov", "EMPLOYEE", "E0001"},
		{"U_LD_001", "ld@careerquest.demo", "Dana Akhmetova", "LD_SPECIALIST", ""},
	}
	for _, account := range accounts {
		hash, err := auth.HashPassword("demo")
		if err != nil {
			return err
		}
		var employeeID any
		if account.employeeID != "" {
			employeeID = account.employeeID
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO users(id,email,name,password_hash,status,application_role_id,employee_id)
			SELECT $1,$2,$3,$4,'ACTIVE',id,$6 FROM application_roles WHERE code=$5
			ON CONFLICT(id) DO NOTHING`, account.id, account.email, account.name, hash, account.role, employeeID); err != nil {
			return err
		}
	}
	return nil
}
