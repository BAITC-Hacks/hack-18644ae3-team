package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Role string

const (
	RoleHR       Role = "hr"
	RoleEmployee Role = "employee"
	RoleLD       Role = "ld"
)

const sessionCookie = "cq_session"

type User struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Role       Role   `json:"role"`
	EmployeeID string `json:"employee_id,omitempty"`
}

type credential struct {
	user         User
	passwordHash [sha256.Size]byte
}

type session struct {
	user      User
	expiresAt time.Time
}

type Service struct {
	mu          sync.RWMutex
	credentials map[string]credential
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

func (s *Service) addDemoUser(user User, password string) {
	key := strings.ToLower(strings.TrimSpace(user.Email))
	s.credentials[key] = credential{user: user, passwordHash: sha256.Sum256([]byte(password))}
}

func (s *Service) Authenticate(email, password string) (User, bool) {
	credential, ok := s.credentials[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		// Keep roughly the same hashing work for unknown and known accounts.
		_ = sha256.Sum256([]byte(password))
		return User{}, false
	}
	candidate := sha256.Sum256([]byte(password))
	if subtle.ConstantTimeCompare(candidate[:], credential.passwordHash[:]) != 1 {
		return User{}, false
	}
	return credential.user, true
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
