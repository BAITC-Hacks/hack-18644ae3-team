package postgres

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"careerquest/internal/career"
	"careerquest/internal/repository"
)

func TestAssessmentRewardsPreserveLevelsAndAreCountedOnce(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL skill reward checks")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := strings.ToLower(randomID("cq_rewards_test_"))
	if _, err := admin.db.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.db.Exec(`DROP SCHEMA ` + schema + ` CASCADE`); err != nil {
			t.Errorf("clean up test schema: %v", err)
		}
	}()
	connectionURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := connectionURL.Query()
	query.Set("search_path", schema)
	connectionURL.RawQuery = query.Encode()
	store, err := Open(ctx, connectionURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = store.db.ExecContext(ctx, `
		INSERT INTO application_roles(id,code) VALUES (1,'LD_SPECIALIST');
		INSERT INTO job_roles(id,name) VALUES (1,'Backend Engineer');
		INSERT INTO grades(id,name,rank) VALUES (1,'Junior',1);
		INSERT INTO skills(id,name,type,category) VALUES ('ADVANCED','Advanced','hard','test'), ('LEARNING','Learning','hard','test');
		INSERT INTO employees(id,full_name,job_role_id,grade_id,last_review_date) VALUES ('TEST','Employee',1,1,CURRENT_DATE-1);
		INSERT INTO employee_skills(employee_id,skill_id,level) VALUES ('TEST','ADVANCED',5), ('TEST','LEARNING',1);
		INSERT INTO users(id,email,name,password_hash,status,application_role_id) VALUES ('ASSESSOR','assessor@example.test','Assessor','unused','ACTIVE',1);
		INSERT INTO events(id,title,type,format,duration_hours) VALUES ('COURSE','Course','course','self_paced',2);
		INSERT INTO event_skill_effects(event_id,skill_id,gain,max_level) VALUES ('COURSE','ADVANCED',1,3), ('COURSE','LEARNING',1,3);
		INSERT INTO enrollments(id,employee_id,event_id,status) VALUES (1,'TEST','COURSE','in_progress');`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.AssessEnrollment(1, "ASSESSOR", repository.AssessmentInput{Result: "PASSED"})
	if err != nil || !result.SkillRewardsApplied {
		t.Fatalf("assess enrollment: %+v, %v", result, err)
	}
	employee, ok := store.Employee("TEST")
	if !ok {
		t.Fatal("employee missing")
	}
	if employee.Skills["ADVANCED"] != 5 || employee.Skills["LEARNING"] != 2 {
		t.Fatalf("incorrect persisted skill rewards: %+v", employee.Skills)
	}
	history := store.ActivitiesForEmployee("TEST")
	if len(history) != 1 || !history[0].SkillRewardsApplied || history[0].Status != "completed" {
		t.Fatalf("completion lost or rewards not marked: %+v", history)
	}
	levels := career.New(store).EffectiveSkills(employee)
	if levels["ADVANCED"] != 5 || levels["LEARNING"] != 2 {
		t.Fatalf("rewards counted twice in recommendation inputs: %+v", levels)
	}
	if _, err := store.AssessEnrollment(1, "ASSESSOR", repository.AssessmentInput{Result: "PASSED"}); err == nil {
		t.Fatal("duplicate reward application accepted")
	}
}
