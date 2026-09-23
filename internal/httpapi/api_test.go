package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"careerquest/internal/career"
	"careerquest/internal/dataset"
	"careerquest/internal/recommendation"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	store, err := dataset.Load(filepath.Join("..", "..", "case_1", "career_quest_dataset"))
	if err != nil {
		t.Fatalf("dataset.Load() error = %v", err)
	}
	careerService := career.New(store)
	return New(store, careerService, recommendation.New(store, careerService))
}

func TestCoreEndpoints(t *testing.T) {
	handler := testHandler(t)
	hrCookie := loginAs(t, handler, "hr@careerquest.demo")
	paths := []string{
		"/employees",
		"/employees/E0001",
		"/employees/E0001/career-path",
		"/employees/E0001/skill-gaps",
		"/employees/E0001/recommendations",
		"/employees/E0001/mandatory-quests",
		"/events/EV_005",
		"/events/EV_005/candidates",
		"/events/EV_005/analytics",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.AddCookie(hrCookie)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, body = %s", path, response.Code, response.Body.String())
			}
		})
	}
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d", response.Code)
	}
}

func TestFrontendAssets(t *testing.T) {
	store, err := dataset.Load(filepath.Join("..", "..", "case_1", "career_quest_dataset"))
	if err != nil {
		t.Fatalf("dataset.Load() error = %v", err)
	}
	careerService := career.New(store)
	handler := NewWithFrontend(store, careerService, recommendation.New(store, careerService), filepath.Join("..", "..", "web"))

	for _, path := range []string{"/", "/styles.css", "/login.js", "/app.js", "/employee.js", "/ld.js"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d", path, response.Code)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), "Career Quest") {
		t.Fatal("frontend index does not contain the application title")
	}

	rolePages := []struct {
		email string
		path  string
	}{
		{"hr@careerquest.demo", "/hr.html"},
		{"employee@careerquest.demo", "/employee.html"},
		{"ld@careerquest.demo", "/ld.html"},
	}
	for _, rolePage := range rolePages {
		cookie := loginAs(t, handler, rolePage.email)
		request = httptest.NewRequest(http.MethodGet, rolePage.path, nil)
		request.AddCookie(cookie)
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s accessing %s status = %d", rolePage.email, rolePage.path, response.Code)
		}
	}

	employeeCookie := loginAs(t, handler, "employee@careerquest.demo")
	request = httptest.NewRequest(http.MethodGet, "/hr.html", nil)
	request.AddCookie(employeeCookie)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("employee accessing HR page status = %d, want 403", response.Code)
	}
}

func TestUpdateCareerGoal(t *testing.T) {
	handler := testHandler(t)
	hrCookie := loginAs(t, handler, "hr@careerquest.demo")
	body, _ := json.Marshal(map[string]string{
		"target_role":  "Product Manager",
		"target_grade": "Junior",
	})
	request := httptest.NewRequest(http.MethodPut, "/employees/E0003/career-goal", bytes.NewReader(body))
	request.AddCookie(hrCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("PUT career goal status = %d, body = %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/employees/E0003/skill-gaps", nil)
	request.AddCookie(hrCookie)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET skill gaps status = %d, body = %s", response.Code, response.Body.String())
	}
	var result struct {
		Basis       string `json:"basis"`
		TargetRole  string `json:"target_role"`
		TargetGrade string `json:"target_grade"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Basis != "career_goal" || result.TargetRole != "Product Manager" || result.TargetGrade != "Junior" {
		t.Fatalf("career goal was not applied: %#v", result)
	}
}

func TestEmployeeCannotAccessAnotherEmployee(t *testing.T) {
	handler := testHandler(t)
	employeeCookie := loginAs(t, handler, "employee@careerquest.demo")

	request := httptest.NewRequest(http.MethodGet, "/employees/E0001", nil)
	request.AddCookie(employeeCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("own profile status = %d, body = %s", response.Code, response.Body.String())
	}

	for _, path := range []string{"/employees", "/employees/E0042", "/employees/E0042/recommendations"} {
		request = httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(employeeCookie)
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("GET %s status = %d, want 403", path, response.Code)
		}
	}

	navigatorBody := `{"employee_id":"E0042","question":"What should I do next?"}`
	request = httptest.NewRequest(http.MethodPost, "/navigator/chat", strings.NewReader(navigatorBody))
	request.AddCookie(employeeCookie)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("navigator for another employee status = %d, want 403", response.Code)
	}
}

func TestProtectedAPIRequiresAuthentication(t *testing.T) {
	handler := testHandler(t)
	for _, path := range []string{"/employees", "/employees/E0001", "/events"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("GET %s status = %d, want 401", path, response.Code)
		}
	}
}

func TestLDCanManageEventsButCannotReadEmployees(t *testing.T) {
	handler := testHandler(t)
	ldCookie := loginAs(t, handler, "ld@careerquest.demo")

	request := httptest.NewRequest(http.MethodGet, "/employees/E0001", nil)
	request.AddCookie(ldCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("L&D employee access status = %d, want 403", response.Code)
	}

	eventBody := `{
		"title":"API Design Lab",
		"description":"Practical API design exercises.",
		"type":"workshop",
		"format":"online",
		"duration_hours":4,
		"mandatory":false,
		"target_roles":["Backend Engineer"],
		"target_grades":["Junior","Middle"],
		"develops_skills":[{"skill_id":"SK_API_DESIGN","gain":1,"max_level":4}],
		"prerequisites":{},
		"upcoming_sessions":["2026-11-15"],
		"learning_link":"https://example.com/api-design"
	}`
	request = httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(eventBody))
	request.AddCookie(ldCookie)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("L&D create event status = %d, body = %s", response.Code, response.Body.String())
	}
}

func loginAs(t *testing.T, handler http.Handler, email string) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": "demo"})
	request := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("login as %s status = %d, body = %s", email, response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("login as %s did not set a session cookie", email)
	}
	return cookies[0]
}
