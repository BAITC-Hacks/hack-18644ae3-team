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
	paths := []string{
		"/health",
		"/employees",
		"/employees/E0001",
		"/employees/E0001/career-path",
		"/employees/E0001/skill-gaps",
		"/employees/E0001/recommendations",
		"/employees/E0001/mandatory-quests",
		"/events/EV_005",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, body = %s", path, response.Code, response.Body.String())
			}
		})
	}
}

func TestFrontendAssets(t *testing.T) {
	store, err := dataset.Load(filepath.Join("..", "..", "case_1", "career_quest_dataset"))
	if err != nil {
		t.Fatalf("dataset.Load() error = %v", err)
	}
	careerService := career.New(store)
	handler := NewWithFrontend(store, careerService, recommendation.New(store, careerService), filepath.Join("..", "..", "web"))

	for _, path := range []string{"/", "/styles.css", "/app.js"} {
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
}

func TestUpdateCareerGoal(t *testing.T) {
	handler := testHandler(t)
	body, _ := json.Marshal(map[string]string{
		"target_role":  "Product Manager",
		"target_grade": "Junior",
	})
	request := httptest.NewRequest(http.MethodPut, "/employees/E0003/career-goal", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("PUT career goal status = %d, body = %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/employees/E0003/skill-gaps", nil)
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
