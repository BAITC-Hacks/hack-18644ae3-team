# Career Quest

Career Quest is a hackathon backend for deterministic, explainable employee
development recommendations. The recommendation engine uses role requirements,
career goals, assessed skills, completed activities, prerequisites, and event
availability. Mandatory activities are never included in recommendations.

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

Career-goal updates are intentionally in-memory and reset when the service is
restarted.

## API

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/health` | Dataset and service status |
| `GET` | `/employees` | Employee summaries for the workspace switcher |
| `GET` | `/employees/{id}` | Employee profile and effective skills |
| `GET` | `/employees/{id}/career-path` | Goal, readiness, and critical blockers |
| `GET` | `/employees/{id}/skill-gaps` | Detailed target skill gaps |
| `GET` | `/employees/{id}/recommendations` | Ranked voluntary activities |
| `GET` | `/employees/{id}/mandatory-quests` | Latest mandatory assignments, kept outside recommendations |
| `PUT` | `/employees/{id}/career-goal` | Set or clear an in-memory career goal |
| `GET` | `/events/{id}` | Event details |
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
