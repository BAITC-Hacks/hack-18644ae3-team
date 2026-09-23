package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleHR       Role = "hr"
	RoleEmployee Role = "employee"
	RoleLD       Role = "ld"
)

const sessionCookie = "cq_session"

type User struct {
	ID                  string `json:"id"`
	Email               string `json:"email"`
	Name                string `json:"name"`
	Role                Role   `json:"role"`
	EmployeeID          string `json:"employee_id,omitempty"`
	RequestedEmployeeID string `json:"requested_employee_id,omitempty"`
	Status              string `json:"status"`
}

type credential struct {
	user         User
	passwordHash string
}

type RegistrationInput struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	EmployeeID string `json:"employee_id,omitempty"`
}

type NewEmployeeApproval struct {
	Role       string `json:"role"`
	Grade      string `json:"grade"`
	Department string `json:"department,omitempty"`
	Team       string `json:"team,omitempty"`
}

type AccountRepository interface {
	FindAccountByEmail(email string) (User, string, error)
	CreatePendingAccount(input RegistrationInput, passwordHash string) (User, error)
	ListRegistrations(status string) ([]User, error)
	UpdateRegistration(userID, status, employeeID string) (User, error)
	ApproveWithNewEmployee(userID string, input NewEmployeeApproval) (User, error)
}

type session struct {
	user      User
	expiresAt time.Time
}

type Service struct {
	mu          sync.RWMutex
	credentials map[string]credential
	accounts    AccountRepository
	sessions    map[string]session
	now         func() time.Time
}

func NewDemoService() *Service {
	s := &Service{
		credentials: make(map[string]credential),
		sessions:    make(map[string]session),
		now:         time.Now,
	}
	s.addDemoUser(User{ID: "U_HR_001", Email: "hr@careerquest.demo", Name: "Aigerim Sarsenova", Role: RoleHR}, "demo")
	s.addDemoUser(User{ID: "U_EMP_001", Email: "employee@careerquest.demo", Name: "Marat Yessenov", Role: RoleEmployee, EmployeeID: "E0001"}, "demo")
	s.addDemoUser(User{ID: "U_LD_001", Email: "ld@careerquest.demo", Name: "Dana Akhmetova", Role: RoleLD}, "demo")
	return s
}

func NewService(accounts AccountRepository) *Service {
	return &Service{accounts: accounts, sessions: make(map[string]session), now: time.Now}
}

func (s *Service) addDemoUser(user User, password string) {
	key := strings.ToLower(strings.TrimSpace(user.Email))
	user.Status = "ACTIVE"
	hash, _ := HashPassword(password)
	s.credentials[key] = credential{user: user, passwordHash: hash}
}

func (s *Service) Authenticate(email, password string) (User, bool) {
	var user User
	var passwordHash string
	if s.accounts != nil {
		var err error
		user, passwordHash, err = s.accounts.FindAccountByEmail(strings.ToLower(strings.TrimSpace(email)))
		if err != nil {
			return User{}, false
		}
	} else {
		credential, ok := s.credentials[strings.ToLower(strings.TrimSpace(email))]
		if !ok {
			return User{}, false
		}
		user, passwordHash = credential.user, credential.passwordHash
	}
	if user.Status != "ACTIVE" || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return User{}, false
	}
	return user, true
}

func HashPassword(password string) (string, error) {
	if len(password) < 8 && password != "demo" {
		return "", errors.New("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func (s *Service) Register(input RegistrationInput) (User, error) {
	if s.accounts == nil {
		return User{}, errors.New("registration is unavailable")
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.EmployeeID = strings.TrimSpace(input.EmployeeID)
	if input.Name == "" || input.Email == "" {
		return User{}, errors.New("name and email are required")
	}
	hash, err := HashPassword(input.Password)
	if err != nil {
		return User{}, err
	}
	return s.accounts.CreatePendingAccount(input, hash)
}

func (s *Service) Registrations(status string) ([]User, error) {
	if s.accounts == nil {
		return []User{}, nil
	}
	return s.accounts.ListRegistrations(status)
}

func (s *Service) DecideRegistration(userID, status, employeeID string) (User, error) {
	if s.accounts == nil {
		return User{}, errors.New("registration management is unavailable")
	}
	if status != "ACTIVE" && status != "REJECTED" {
		return User{}, errors.New("status must be ACTIVE or REJECTED")
	}
	return s.accounts.UpdateRegistration(userID, status, strings.TrimSpace(employeeID))
}

func (s *Service) ApproveNewRegistration(userID string, input NewEmployeeApproval) (User, error) {
	if s.accounts == nil {
		return User{}, errors.New("registration management is unavailable")
	}
	input.Role = strings.TrimSpace(input.Role)
	input.Grade = strings.TrimSpace(input.Grade)
	input.Department = strings.TrimSpace(input.Department)
	input.Team = strings.TrimSpace(input.Team)
	if input.Role == "" || input.Grade == "" {
		return User{}, errors.New("job role and grade are required for a new employee")
	}
	return s.accounts.ApproveWithNewEmployee(userID, input)
}

func (s *Service) StartSession(w http.ResponseWriter, r *http.Request, user User) error {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	expires := s.now().Add(8 * time.Hour)
	s.mu.Lock()
	s.sessions[token] = session{user: user, expiresAt: expires}
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int((8 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *Service) EndSession(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		s.mu.Lock()
		delete(s.sessions, cookie.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Service) UserFromRequest(r *http.Request) (User, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return User{}, false
	}
	s.mu.RLock()
	session, ok := s.sessions[cookie.Value]
	s.mu.RUnlock()
	if !ok || !session.expiresAt.After(s.now()) {
		if ok {
			s.mu.Lock()
			delete(s.sessions, cookie.Value)
			s.mu.Unlock()
		}
		return User{}, false
	}
	return session.user, true
}

type contextKey struct{}

func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(contextKey{}).(User)
	return user, ok
}

func (s *Service) RequireAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.UserFromRequest(r)
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, user)))
	})
}

func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	allowed := roleSet(roles)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok || !allowed[user.Role] {
				writeAuthError(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireEmployeeOwnerOrRole(pathParameter string, roles ...Role) func(http.Handler) http.Handler {
	allowed := roleSet(roles)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				writeAuthError(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			if allowed[user.Role] || (user.Role == RoleEmployee && user.EmployeeID != "" && user.EmployeeID == r.PathValue(pathParameter)) {
				next.ServeHTTP(w, r)
				return
			}
			writeAuthError(w, http.StatusForbidden, "you cannot access another employee's information")
		})
	}
}

func CanAccessEmployee(user User, employeeID string, roles ...Role) bool {
	if roleSet(roles)[user.Role] {
		return true
	}
	return user.Role == RoleEmployee && user.EmployeeID != "" && user.EmployeeID == employeeID
}

func roleSet(roles []Role) map[Role]bool {
	result := make(map[Role]bool, len(roles))
	for _, role := range roles {
		result[role] = true
	}
	return result
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
