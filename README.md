# Career Quest

Career Quest is a role-based career development application for employees, HR,
and Learning & Development specialists. Recommendations remain deterministic
and explainable: the engine uses role requirements, career goals, assessed
skills, completed activities, prerequisites, and event availability.

## Run

The service uses only the Go standard library and loads the supplied dataset
into memory.

```powershell
go run ./cmd/server
```

The default address is `http://localhost:8081`. Both settings can be changed:

```powershell
go run ./cmd/server -addr :8090 -data case_1/career_quest_dataset
```

Open `http://localhost:8081` in a browser to use the Career Quest interface.
The frontend is served by the same Go process, so there is no Node.js install or
separate development server. Frontend files live in `web/`; use `-web` to point
the server at another asset directory.

Career-goal and course/event updates are intentionally in-memory and reset when
the service is restarted.

## Demo accounts

All demo accounts use password `demo`.

| Role | Email | Access |
|---|---|---|
| HR | `hr@careerquest.demo` | Employee directory, employee development data, event candidates, organization analytics |
| Employee | `employee@careerquest.demo` | Only employee `E0001` and that employee's career journey |
| L&D Specialist | `ld@careerquest.demo` | Course/event creation, editing, sessions, links, and aggregate course analytics |

Authentication uses an HTTP-only, same-site session cookie. API authorization
is enforced by role and, for employee routes, by employee ownership.

## API

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/health` | Dataset and service status |
| `POST` | `/auth/login` | Start a role-scoped session |
| `GET` | `/auth/me` | Current authenticated user |
| `POST` | `/auth/logout` | End the current session |
| `GET` | `/catalog` | Skills, roles, grades, and proficiency scale |
| `GET` | `/employees` | HR-only searchable employee directory |
| `GET` | `/employees/{id}` | HR or owning employee profile |
| `GET` | `/employees/{id}/career-path` | Goal, readiness, and critical blockers |
| `GET` | `/employees/{id}/skill-gaps` | Detailed target skill gaps |
| `GET` | `/employees/{id}/recommendations` | Ranked voluntary activities |
| `GET` | `/employees/{id}/mandatory-quests` | Latest mandatory assignments, kept outside recommendations |
| `GET` | `/employees/{id}/activities` | Joined development activity history |
| `PUT` | `/employees/{id}/career-goal` | Set or clear an in-memory career goal |
| `GET` | `/events` | Authenticated activity catalog |
| `POST` | `/events` | L&D-only activity creation |
| `GET` | `/events/{id}` | Event details |
| `PUT` | `/events/{id}` | L&D-only activity update |
| `GET` | `/events/{id}/candidates` | HR-only ranked employee candidates |
| `GET` | `/events/{id}/analytics` | HR/L&D aggregate activity analytics |
| `POST` | `/navigator/chat` | Deterministic explanation of a recommendation |

Set a career goal:

```json
{
  "target_role": "Backend Engineer",
  "target_grade": "Middle"
}
```

Clear a career goal:

```json
{
  "clear": true
}
```

The Navigator endpoint currently returns an explanation generated from the
recommendation facts. It does not call an LLM and does not choose activities.

## Verify

```powershell
go test ./...
go vet ./...
go build ./cmd/server
```
