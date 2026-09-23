package postgres

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"careerquest/internal/auth"
)

// Uses an isolated schema so regression checks never change real accounts.
func TestRegistrationPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL registration checks")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := strings.ToLower(randomID("cq_registration_test_"))
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

	// Simulate an existing installation with linked pending accounts before 002.
	initial, err := migrationFiles.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, string(initial)); err != nil {
		t.Fatal(err)
	}
	_, err = store.db.ExecContext(ctx, `
		CREATE TABLE schema_migrations(name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
		INSERT INTO schema_migrations(name) VALUES ('001_init.sql');
		INSERT INTO application_roles(id,code) VALUES (1,'EMPLOYEE');
		INSERT INTO job_roles(id,name) VALUES (1,'Backend Engineer');
		INSERT INTO grades(id,name,rank) VALUES (1,'Junior',1);
		INSERT INTO employees(id,full_name,job_role_id,grade_id) VALUES
			('ACTIVE_EMP','Active employee',1,1), ('PENDING_EMP','Pending employee',1,1), ('REJECTED_EMP','Rejected employee',1,1);
		INSERT INTO users(id,email,name,password_hash,status,application_role_id,employee_id) VALUES
			('ACTIVE_USER','active@example.test','Active','unused','ACTIVE',1,'ACTIVE_EMP'),
			('PENDING_USER','pending@example.test','Pending','unused','PENDING',1,'PENDING_EMP'),
			('REJECTED_USER','rejected@example.test','Rejected','unused','REJECTED',1,'REJECTED_EMP');`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := store.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	// A compliance course can legitimately have no skill improvement or session.
	if _, err := store.db.ExecContext(ctx, `INSERT INTO events(id,title,type,format,duration_hours) VALUES('NO_SKILLS','Compliance','compliance','online',1)`); err != nil {
		t.Fatal(err)
	}
	compliance, ok := store.Event("NO_SKILLS")
	if !ok {
		t.Fatal("could not load compliance course")
	}
	encoded, err := json.Marshal(compliance)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"develops_skills":[]`, `"upcoming_sessions":[]`, `"target_roles":[]`, `"target_grades":[]`} {
		if !strings.Contains(string(encoded), field) {
			t.Fatalf("event has a null array %s: %s", field, encoded)
		}
	}
	for _, status := range []string{"PENDING", "REJECTED"} {
		users, err := store.ListRegistrations(status)
		if err != nil || len(users) != 1 || users[0].EmployeeID != "" || users[0].RequestedEmployeeID != status+"_EMP" {
			t.Fatalf("migrate %s: users=%+v error=%v", status, users, err)
		}
	}
	active, _, err := store.FindAccountByEmail("active@example.test")
	if err != nil || active.EmployeeID != "ACTIVE_EMP" {
		t.Fatalf("migration changed active account: %+v, %v", active, err)
	}

	service := auth.NewService(store)
	for _, requestedID := range []string{"", "DOES_NOT_EXIST", "ACTIVE_EMP", "PENDING_EMP"} {
		t.Run("requested_"+requestedID, func(t *testing.T) {
			input := auth.RegistrationInput{Name: "New applicant", Email: requestedID + "new@example.test", Password: "test-password", EmployeeID: " " + requestedID + " "}
			user, err := service.Register(input)
			if err != nil {
				t.Fatalf("registration should succeed without linking an employee: %v", err)
			}
			if user.Status != "PENDING" || user.Role != auth.RoleEmployee || user.EmployeeID != "" || user.RequestedEmployeeID != requestedID {
				t.Fatalf("unexpected registration: %+v", user)
			}
			if _, ok := service.Authenticate(input.Email, input.Password); ok {
				t.Fatal("pending account can log in")
			}
			if _, err := service.Register(input); err == nil || !strings.Contains(err.Error(), "email already exists") {
				t.Fatalf("duplicate email should have a readable error: %v", err)
			}
			if requestedID != "DOES_NOT_EXIST" {
				return
			}
			for _, invalidID := range []string{"", "DOES_NOT_EXIST", "ACTIVE_EMP"} {
				if _, err := service.DecideRegistration(user.ID, "ACTIVE", invalidID); err == nil || strings.Contains(err.Error(), "SQLSTATE") {
					t.Fatalf("invalid approval %q must fail clearly: %v", invalidID, err)
				}
				unchanged, _, err := store.FindAccountByEmail(input.Email)
				if err != nil || unchanged.Status != "PENDING" || unchanged.EmployeeID != "" {
					t.Fatalf("failed approval changed account: %+v, %v", unchanged, err)
				}
			}
			approved, err := service.DecideRegistration(user.ID, "ACTIVE", "PENDING_EMP")
			if err != nil || approved.EmployeeID != "PENDING_EMP" || approved.Status != "ACTIVE" {
				t.Fatalf("HR approval failed: %+v, %v", approved, err)
			}
			if _, ok := service.Authenticate(input.Email, input.Password); !ok {
				t.Fatal("approved account cannot log in")
			}
		})
	}
}
