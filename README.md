# Career Quest

Career Quest is a role-based career development application for employees, HR,
and Learning & Development specialists. PostgreSQL is the system of record;
the files in `case_1/career_quest_dataset` are imported only as idempotent seed
data. Recommendations remain deterministic and explainable.

## Run locally

Requirements: Go 1.23+ and Docker Desktop (or another PostgreSQL 16 instance).

```powershell
docker compose up -d db
go run ./cmd/server
```

Open `http://localhost:8081`. The server automatically applies embedded SQL
migrations and safely seeds the supplied dataset on startup. Starting it again
does not duplicate employees, events, enrollments, or history.

The repository PostgreSQL container uses host port `55432` to avoid conflicts
with locally installed PostgreSQL. Override the connection when needed:

```powershell
$env:DATABASE_URL = "postgres://user:password@localhost:5432/careerquest?sslmode=disable"
go run ./cmd/server -addr :8090
```

Useful flags:

- `-data case_1/career_quest_dataset` selects seed files.
- `-seed=false` skips the seed pass after migrations.
- `-database-url ...` overrides `DATABASE_URL`.
- `-addr :8090` changes the HTTP port if 8081 is occupied.

## Demo accounts

All demo accounts use password `demo`.

| Role | Email | Access |
|---|---|---|
| HR | `hr@careerquest.demo` | Employee directory, registrations, profiles, candidates, analytics |
| Employee | `employee@careerquest.demo` | Only employee `E0001`, personal career data, activity catalog |
| L&D Specialist | `ld@careerquest.demo` | Events, sessions, participants, assessments, aggregate analytics |

New accounts can be requested from the sign-in page. They remain `PENDING`
until HR links the account to an employee and approves it.
The optional employee ID entered during registration is saved for HR to verify;
it does not need to exist yet and does not grant access to that employee.
HR can create the employee record first, then select it when approving.

## Main API

| Method | Path | Access / purpose |
|---|---|---|
| `POST` | `/auth/register` | Public; create a pending employee account |
| `POST` | `/auth/login` | Active users; start a session |
| `GET` | `/auth/me` | Authenticated user |
| `POST` | `/auth/logout` | End session |
| `GET` | `/registrations` | HR; pending registration requests |
| `POST` | `/registrations/{id}/approve` | HR; activate and link to an employee |
| `POST` | `/registrations/{id}/reject` | HR; reject request |
| `GET` | `/employees?q=&department=&team=&role=&grade=` | HR; database employee search |
| `POST` | `/employees` | HR; create employee |
| `GET` | `/employees/{id}` | HR or owning employee |
| `GET` | `/employees/{id}/career-path` | HR or owning employee |
| `GET` | `/employees/{id}/skill-gaps` | HR or owning employee |
| `GET` | `/employees/{id}/recommendations` | HR or owning employee |
| `GET` | `/employees/{id}/activities` | HR or owning employee |
| `PUT` | `/employees/{id}/career-goal` | HR or owning employee |
| `GET` | `/events?q=&type=&role=&grade=` | Authenticated; database catalog search |
| `POST` | `/events` | L&D; create activity |
| `PUT` | `/events/{id}` | L&D; update activity and sessions |
| `GET` | `/events/{id}/candidates` | HR; ranked candidates |
| `GET` | `/events/{id}/analytics` | HR or L&D; aggregates |
| `GET` | `/events/{id}/participants` | L&D; enrolled participants |
| `POST` | `/enrollments/{id}/assessment` | L&D; pass/fail, score, feedback, skill reward |
| `POST` | `/navigator/chat` | HR or owning employee; deterministic explanation |

Authentication uses an HTTP-only same-site session cookie. Authorization is
enforced by reusable backend middleware. Employee ownership is checked on the
server, so changing an employee ID in a URL returns `403 Forbidden`.

## Verify

```powershell
gofmt -w ./cmd ./internal
go test ./...
go build ./...
```

The current Navigator explains deterministic recommendation facts. It does not
yet call an external LLM and does not choose activities itself.
