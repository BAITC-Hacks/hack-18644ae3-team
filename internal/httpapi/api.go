package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"careerquest/internal/career"
	"careerquest/internal/dataset"
	"careerquest/internal/domain"
	"careerquest/internal/recommendation"
)

type API struct {
	store          *dataset.Store
	career         *career.Service
	recommendation *recommendation.Service
}

func New(store *dataset.Store, careerService *career.Service, recommendationService *recommendation.Service) http.Handler {
	return newHandler(store, careerService, recommendationService, "")
}

func NewWithFrontend(store *dataset.Store, careerService *career.Service, recommendationService *recommendation.Service, webDir string) http.Handler {
	return newHandler(store, careerService, recommendationService, webDir)
}

func newHandler(store *dataset.Store, careerService *career.Service, recommendationService *recommendation.Service, webDir string) http.Handler {
	api := &API{store: store, career: careerService, recommendation: recommendationService}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("GET /employees", api.employees)
	mux.HandleFunc("GET /employees/{id}", api.employee)
	mux.HandleFunc("GET /employees/{id}/career-path", api.careerPath)
	mux.HandleFunc("GET /employees/{id}/skill-gaps", api.skillGaps)
	mux.HandleFunc("GET /employees/{id}/recommendations", api.recommendations)
	mux.HandleFunc("GET /employees/{id}/mandatory-quests", api.mandatoryQuests)
	mux.HandleFunc("PUT /employees/{id}/career-goal", api.updateCareerGoal)
	mux.HandleFunc("GET /events/{id}", api.event)
	mux.HandleFunc("POST /navigator/chat", api.navigator)
	if webDir != "" {
		mux.Handle("GET /", http.FileServer(http.Dir(webDir)))
	}
	return cors(mux)
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"dataset":    a.store.Meta().Dataset,
		"version":    a.store.Meta().Version,
		"as_of_date": a.store.Meta().AsOfDate,
	})
}

type employeeSummary struct {
	ID                string             `json:"employee_id"`
	FullName          string             `json:"full_name"`
	Department        string             `json:"department"`
	Role              string             `json:"role"`
	Grade             string             `json:"grade"`
	PreferredLanguage string             `json:"preferred_language"`
	CareerGoal        *domain.CareerGoal `json:"career_goal"`
}

func (a *API) employees(w http.ResponseWriter, _ *http.Request) {
	employees := a.store.Employees()
	result := make([]employeeSummary, 0, len(employees))
	for _, employee := range employees {
		result = append(result, employeeSummary{
			ID:                employee.ID,
			FullName:          employee.FullName,
			Department:        employee.Department,
			Role:              employee.Role,
			Grade:             employee.Grade,
			PreferredLanguage: employee.PreferredLanguage,
			CareerGoal:        employee.CareerGoal,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"employees": result})
}

func (a *API) employee(w http.ResponseWriter, r *http.Request) {
	employee, ok := a.store.Employee(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "employee not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"employee":         employee,
		"effective_skills": a.career.EffectiveSkills(employee),
	})
}

func (a *API) skillGaps(w http.ResponseWriter, r *http.Request) {
	assessment, err := a.career.Assess(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, assessment)
}

func (a *API) careerPath(w http.ResponseWriter, r *http.Request) {
	employee, ok := a.store.Employee(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "employee not found")
		return
	}
	assessment, err := a.career.AssessEmployee(employee)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	critical := make([]domain.SkillGap, 0)
	for _, gap := range assessment.Gaps {
		if gap.Critical {
			critical = append(critical, gap)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"employee_id":       employee.ID,
		"current":           map[string]string{"role": employee.Role, "grade": employee.Grade},
		"career_goal":       employee.CareerGoal,
		"basis":             assessment.Basis,
		"target":            map[string]string{"role": assessment.TargetRole, "grade": assessment.TargetGrade},
		"readiness_percent": assessment.ReadinessPercent,
		"skill_gap_count":   len(assessment.Gaps),
		"critical_blockers": critical,
		"goal_required":     employee.CareerGoal == nil,
	})
}

func (a *API) recommendations(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.store.Employee(r.PathValue("id")); !ok {
		writeError(w, http.StatusNotFound, "employee not found")
		return
	}
	result, err := a.recommendation.ForEmployee(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type mandatoryQuest struct {
	EventID       string  `json:"event_id"`
	Title         string  `json:"title"`
	Type          string  `json:"type"`
	Format        string  `json:"format"`
	DurationHours float64 `json:"duration_hours"`
	Status        string  `json:"status"`
	CompletionPct int     `json:"completion_pct"`
	AssignedDate  string  `json:"assigned_date"`
	DueDate       string  `json:"due_date,omitempty"`
}

func (a *API) mandatoryQuests(w http.ResponseWriter, r *http.Request) {
	employeeID := r.PathValue("id")
	if _, ok := a.store.Employee(employeeID); !ok {
		writeError(w, http.StatusNotFound, "employee not found")
		return
	}
	latest := make(map[string]domain.Activity)
	for _, activity := range a.store.ActivitiesForEmployee(employeeID) {
		event, ok := a.store.Event(activity.EventID)
		if !ok || !event.Mandatory {
			continue
		}
		previous, exists := latest[activity.EventID]
		if !exists || activity.Date > previous.Date || (activity.Date == previous.Date && activity.RecordID > previous.RecordID) {
			latest[activity.EventID] = activity
		}
	}
	quests := make([]mandatoryQuest, 0, len(latest))
	for eventID, activity := range latest {
		event, _ := a.store.Event(eventID)
		quests = append(quests, mandatoryQuest{
			EventID:       event.ID,
			Title:         event.Title,
			Type:          event.Type,
			Format:        event.Format,
			DurationHours: event.DurationHours,
			Status:        activity.Status,
			CompletionPct: activity.CompletionPct,
			AssignedDate:  activity.Date,
			DueDate:       activity.DueDate,
		})
	}
	slices.SortFunc(quests, func(a, b mandatoryQuest) int {
		if a.Status == "overdue" && b.Status != "overdue" {
			return -1
		}
		if b.Status == "overdue" && a.Status != "overdue" {
			return 1
		}
		return strings.Compare(a.EventID, b.EventID)
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"employee_id": employeeID,
		"quests":      quests,
	})
}

type goalUpdateRequest struct {
	TargetRole  string `json:"target_role"`
	TargetGrade string `json:"target_grade"`
	Clear       bool   `json:"clear"`
}

func (a *API) updateCareerGoal(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.store.Employee(r.PathValue("id")); !ok {
		writeError(w, http.StatusNotFound, "employee not found")
		return
	}
	var request goalUpdateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var goal *domain.CareerGoal
	if !request.Clear {
		if request.TargetRole == "" || request.TargetGrade == "" {
			writeError(w, http.StatusBadRequest, "target_role and target_grade are required")
			return
		}
		goal = &domain.CareerGoal{TargetRole: request.TargetRole, TargetGrade: request.TargetGrade}
	}
	employee, err := a.store.UpdateCareerGoal(r.PathValue("id"), goal)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	assessment, err := a.career.AssessEmployee(employee)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"employee_id": employee.ID,
		"career_goal": employee.CareerGoal,
		"assessment":  assessment,
	})
}

func (a *API) event(w http.ResponseWriter, r *http.Request) {
	event, ok := a.store.Event(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "event not found")
		return
	}
	writeJSON(w, http.StatusOK, event)
}

type navigatorRequest struct {
	EmployeeID string `json:"employee_id"`
	EventID    string `json:"event_id,omitempty"`
	Question   string `json:"question,omitempty"`
}

func (a *API) navigator(w http.ResponseWriter, r *http.Request) {
	var request navigatorRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if request.EmployeeID == "" {
		writeError(w, http.StatusBadRequest, "employee_id is required")
		return
	}
	if _, ok := a.store.Employee(request.EmployeeID); !ok {
		writeError(w, http.StatusNotFound, "employee not found")
		return
	}
	result, err := a.recommendation.ForEmployee(request.EmployeeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var selected *recommendation.Item
	if request.EventID != "" {
		for i := range result.Recommendations {
			if result.Recommendations[i].EventID == request.EventID {
				selected = &result.Recommendations[i]
				break
			}
		}
		if selected == nil {
			writeError(w, http.StatusBadRequest, "event is not currently recommended for this employee")
			return
		}
	} else if len(result.Recommendations) > 0 {
		selected = &result.Recommendations[0]
	}
	if selected == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"mode":        "deterministic",
			"employee_id": request.EmployeeID,
			"message":     "No eligible skill-building activity currently matches the employee's gaps.",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"mode":           "deterministic",
		"employee_id":    request.EmployeeID,
		"event_id":       selected.EventID,
		"message":        selected.Explanation,
		"recommendation": selected,
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("invalid JSON body: only one object is allowed")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, OPTIONS")
		if strings.EqualFold(r.Method, http.MethodOptions) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
