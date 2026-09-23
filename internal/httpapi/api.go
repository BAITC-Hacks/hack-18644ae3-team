package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"careerquest/internal/assessment"
	"careerquest/internal/auth"
	"careerquest/internal/career"
	"careerquest/internal/domain"
	"careerquest/internal/events"
	"careerquest/internal/navigator"
	"careerquest/internal/recommendation"
	"careerquest/internal/repository"
)

type API struct {
	store            repository.Store
	auth             *auth.Service
	career           *career.Service
	events           *events.Service
	navigatorService *navigator.Service
	recommendation   *recommendation.Service
	assessment       *assessment.Service
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var request auth.RegistrationInput
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := a.auth.Register(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"user": user, "message": "Registration submitted for HR approval."})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, ok := a.auth.Authenticate(request.Email, request.Password)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err := a.auth.StartSession(w, r, user); err != nil {
		writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "redirect": roleHome(user.Role)})
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "redirect": roleHome(user.Role)})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	a.auth.EndSession(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"logged_out": true})
}

func roleHome(role auth.Role) string {
	switch role {
	case auth.RoleHR:
		return "/hr.html"
	case auth.RoleEmployee:
		return "/employee.html"
	case auth.RoleLD:
		return "/ld.html"
	default:
		return "/"
	}
}

func New(store repository.Store, careerService *career.Service, recommendationService *recommendation.Service) http.Handler {
	return newHandler(store, careerService, recommendationService, auth.NewDemoService(), "")
}

func NewWithFrontend(store repository.Store, careerService *career.Service, recommendationService *recommendation.Service, webDir string) http.Handler {
	return newHandler(store, careerService, recommendationService, auth.NewDemoService(), webDir)
}

func NewWithAuth(store repository.Store, careerService *career.Service, recommendationService *recommendation.Service, authService *auth.Service, webDir string) http.Handler {
	return newHandler(store, careerService, recommendationService, authService, webDir)
}

func newHandler(store repository.Store, careerService *career.Service, recommendationService *recommendation.Service, authService *auth.Service, webDir string) http.Handler {
	api := &API{store: store, auth: authService, career: careerService, events: events.New(store), navigatorService: navigator.New(store, careerService, recommendationService), recommendation: recommendationService, assessment: assessment.New(store)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("POST /auth/login", api.login)
	mux.HandleFunc("POST /auth/register", api.register)
	mux.Handle("GET /auth/me", authService.RequireAuthenticated(http.HandlerFunc(api.me)))
	mux.Handle("POST /auth/logout", authService.RequireAuthenticated(http.HandlerFunc(api.logout)))

	requireHR := func(handler http.HandlerFunc) http.Handler {
		return authService.RequireAuthenticated(auth.RequireRole(auth.RoleHR)(handler))
	}
	requireLD := func(handler http.HandlerFunc) http.Handler {
		return authService.RequireAuthenticated(auth.RequireRole(auth.RoleLD)(handler))
	}
	requireHRorLD := func(handler http.HandlerFunc) http.Handler {
		return authService.RequireAuthenticated(auth.RequireRole(auth.RoleHR, auth.RoleLD)(handler))
	}
	requireEmployeeAccess := func(handler http.HandlerFunc) http.Handler {
		return authService.RequireAuthenticated(auth.RequireEmployeeOwnerOrRole("id", auth.RoleHR)(handler))
	}

	mux.Handle("GET /catalog", authService.RequireAuthenticated(http.HandlerFunc(api.catalog)))
	mux.Handle("GET /employees", requireHR(http.HandlerFunc(api.employees)))
	mux.Handle("POST /employees", requireHR(http.HandlerFunc(api.createEmployee)))
	mux.Handle("GET /registrations", requireHR(http.HandlerFunc(api.registrations)))
	mux.Handle("POST /registrations/{id}/approve", requireHR(http.HandlerFunc(api.approveRegistration)))
	mux.Handle("POST /registrations/{id}/reject", requireHR(http.HandlerFunc(api.rejectRegistration)))
	mux.Handle("GET /employees/{id}", requireEmployeeAccess(http.HandlerFunc(api.employee)))
	mux.Handle("GET /employees/{id}/career-path", requireEmployeeAccess(http.HandlerFunc(api.careerPath)))
	mux.Handle("GET /employees/{id}/skill-gaps", requireEmployeeAccess(http.HandlerFunc(api.skillGaps)))
	mux.Handle("GET /employees/{id}/recommendations", requireEmployeeAccess(http.HandlerFunc(api.recommendations)))
	mux.Handle("GET /employees/{id}/mandatory-quests", requireEmployeeAccess(http.HandlerFunc(api.mandatoryQuests)))
	mux.Handle("GET /employees/{id}/activities", requireEmployeeAccess(http.HandlerFunc(api.employeeActivities)))
	mux.Handle("PUT /employees/{id}/career-goal", requireEmployeeAccess(http.HandlerFunc(api.updateCareerGoal)))
	mux.Handle("GET /events", authService.RequireAuthenticated(http.HandlerFunc(api.listEvents)))
	mux.Handle("POST /events", requireLD(http.HandlerFunc(api.createEvent)))
	mux.Handle("GET /events/{id}", authService.RequireAuthenticated(http.HandlerFunc(api.event)))
	mux.Handle("PUT /events/{id}", requireLD(http.HandlerFunc(api.updateEvent)))
	mux.Handle("GET /events/{id}/candidates", requireHR(http.HandlerFunc(api.eventCandidates)))
	mux.Handle("GET /events/{id}/analytics", requireHRorLD(http.HandlerFunc(api.eventAnalytics)))
	mux.Handle("GET /events/{id}/participants", requireLD(http.HandlerFunc(api.eventParticipants)))
	mux.Handle("POST /enrollments/{id}/assessment", requireLD(http.HandlerFunc(api.assessEnrollment)))
	mux.Handle("POST /navigator/chat", authService.RequireAuthenticated(http.HandlerFunc(api.navigator)))
	if webDir != "" {
		servePage := func(name string) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, filepath.Join(webDir, name))
			})
		}
		mux.Handle("GET /hr.html", authService.RequireAuthenticated(auth.RequireRole(auth.RoleHR)(servePage("hr.html"))))
		mux.Handle("GET /employee.html", authService.RequireAuthenticated(auth.RequireRole(auth.RoleEmployee)(servePage("employee.html"))))
		mux.Handle("GET /ld.html", authService.RequireAuthenticated(auth.RequireRole(auth.RoleLD)(servePage("ld.html"))))
		mux.Handle("GET /", http.FileServer(http.Dir(webDir)))
	}
	return cors(mux)
}

func (a *API) registrations(w http.ResponseWriter, r *http.Request) {
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if status == "" {
		status = "PENDING"
	}
	users, err := a.auth.Registrations(status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load registrations")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"registrations": users})
}

type registrationDecision struct {
	EmployeeID string `json:"employee_id"`
}

func (a *API) approveRegistration(w http.ResponseWriter, r *http.Request) {
	var request registrationDecision
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := a.auth.DecideRegistration(r.PathValue("id"), "ACTIVE", request.EmployeeID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (a *API) rejectRegistration(w http.ResponseWriter, r *http.Request) {
	user, err := a.auth.DecideRegistration(r.PathValue("id"), "REJECTED", "")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
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
	Email             string             `json:"email,omitempty"`
	Department        string             `json:"department"`
	Team              string             `json:"team,omitempty"`
	Role              string             `json:"role"`
	Grade             string             `json:"grade"`
	PreferredLanguage string             `json:"preferred_language"`
	CareerGoal        *domain.CareerGoal `json:"career_goal"`
}

func (a *API) catalog(w http.ResponseWriter, _ *http.Request) {
	profiles := a.store.Profiles()
	roles, grades := make(map[string]bool), make(map[string]bool)
	for _, profile := range profiles {
		roles[profile.Role] = true
		grades[profile.Grade] = true
	}
	roleList, gradeList := make([]string, 0, len(roles)), make([]string, 0, len(grades))
	for role := range roles {
		roleList = append(roleList, role)
	}
	for grade := range grades {
		gradeList = append(gradeList, grade)
	}
	slices.Sort(roleList)
	gradeOrder := map[string]int{"Junior": 0, "Middle": 1, "Senior": 2, "Lead": 3}
	slices.SortFunc(gradeList, func(a, b string) int { return gradeOrder[a] - gradeOrder[b] })
	writeJSON(w, http.StatusOK, map[string]any{
		"skills":            a.store.Skills(),
		"roles":             roleList,
		"grades":            gradeList,
		"proficiency_scale": a.store.ProficiencyScale(),
	})
}

func (a *API) employees(w http.ResponseWriter, r *http.Request) {
	employees, err := a.store.SearchEmployees(repository.EmployeeSearch{
		Query:      r.URL.Query().Get("q"),
		Department: r.URL.Query().Get("department"),
		Team:       r.URL.Query().Get("team"),
		Role:       r.URL.Query().Get("role"),
		Grade:      r.URL.Query().Get("grade"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not search employees")
		return
	}
	result := make([]employeeSummary, 0, len(employees))
	for _, employee := range employees {
		result = append(result, employeeSummary{
			ID:                employee.ID,
			FullName:          employee.FullName,
			Email:             employee.Email,
			Department:        employee.Department,
			Team:              employee.Team,
			Role:              employee.Role,
			Grade:             employee.Grade,
			PreferredLanguage: employee.PreferredLanguage,
			CareerGoal:        employee.CareerGoal,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"employees": result})
}

func (a *API) createEmployee(w http.ResponseWriter, r *http.Request) {
	manager, ok := a.store.(repository.EmployeeManager)
	if !ok {
		writeError(w, http.StatusNotImplemented, "employee creation is unavailable")
		return
	}
	var request repository.EmployeeCreate
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	employee, err := manager.CreateEmployee(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"employee": employee})
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

func (a *API) employeeActivities(w http.ResponseWriter, r *http.Request) {
	activities, err := a.events.EmployeeActivities(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"employee_id": r.PathValue("id"),
		"activities":  activities,
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

func (a *API) listEvents(w http.ResponseWriter, r *http.Request) {
	events, err := a.store.SearchEvents(repository.EventSearch{Query: r.URL.Query().Get("q"), Type: r.URL.Query().Get("type"), Role: r.URL.Query().Get("role"), Grade: r.URL.Query().Get("grade")})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not search activities")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (a *API) createEvent(w http.ResponseWriter, r *http.Request) {
	var event domain.Event
	if err := decodeJSON(w, r, &event); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	created, err := a.store.CreateEvent(event)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *API) updateEvent(w http.ResponseWriter, r *http.Request) {
	var event domain.Event
	if err := decodeJSON(w, r, &event); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, err := a.store.UpdateEvent(r.PathValue("id"), event)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *API) eventCandidates(w http.ResponseWriter, r *http.Request) {
	candidates, err := a.recommendation.CandidatesForEvent(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"event_id":   r.PathValue("id"),
		"candidates": candidates,
	})
}

func (a *API) eventAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics, err := a.events.Analytics(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, analytics)
}

func (a *API) eventParticipants(w http.ResponseWriter, r *http.Request) {
	participants, err := a.assessment.Participants(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load participants")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"participants": participants})
}

func (a *API) assessEnrollment(w http.ResponseWriter, r *http.Request) {
	enrollmentID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || enrollmentID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid enrollment id")
		return
	}
	var request repository.AssessmentInput
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	result, err := a.assessment.Assess(enrollmentID, user.ID, request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assessment": result})
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
	user, _ := auth.UserFromContext(r.Context())
	if !auth.CanAccessEmployee(user, request.EmployeeID, auth.RoleHR) {
		writeError(w, http.StatusForbidden, "you cannot access another employee's information")
		return
	}
	if _, ok := a.store.Employee(request.EmployeeID); !ok {
		writeError(w, http.StatusNotFound, "employee not found")
		return
	}
	answer, err := a.navigatorService.Answer(request.EmployeeID, request.EventID, request.Question)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, answer)
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
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		if strings.EqualFold(r.Method, http.MethodOptions) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
