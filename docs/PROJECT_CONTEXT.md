# Career Quest

Career Quest is a gamified employee career development platform.

## Main idea

Employees receive many disconnected HR activities:
- courses
- certifications
- workshops
- rotations
- assessments

Career Quest connects these activities into a visible career path.

Example:

Backend Junior
    ↓
System Design Fundamentals
    ↓
API Design 2 → 3
System Design 1 → 2
    ↓
Backend Middle

## Dataset

Dataset contains:
- employees
- employee skills
- role / grade requirements
- events
- event prerequisites
- activity history
- career goals

## Core logic

The system should NOT randomly assign events.

For voluntary events:

1. Event is created / announced.
2. Career Quest checks employees.
3. Filter by:
   - role
   - grade
   - prerequisites
   - previous activity
4. Compare event skills with employee skill gaps.
5. Rank relevant events/employees.
6. Recommend the event.

Mandatory events are assigned by HR and shouldn't be part of the
recommendation algorithm.

## Main feature

Career Quest should answer:

1. Why should I complete this activity?
2. What skills will it improve?
3. How does it bring me closer to my career goal?

## Recommendation engine

Employee:
- current role
- current grade
- career goal
- skills
- history

Event:
- target roles
- target grades
- develops_skills
- prerequisites
- duration
- upcoming sessions

Output:
- match score
- skill gaps covered
- career readiness impact
- explanation

## Example

Employee:
Backend Engineer Junior

Goal:
Backend Engineer Middle

Current:
API Design = 2
System Design = 1

Required:
API Design = 3
System Design = 2

Event:
System Design Fundamentals

Effects:
API Design +1
System Design +1

Therefore this event should be highly recommended.

## Product concepts

- Main Quest
- Side Quest
- Locked Quest
- Mandatory Quest
- Completed Quest
- Career Readiness %
- Skill Tree
- AI Navigator

## AI Navigator

LLM should NOT make the recommendation itself.

Backend recommendation engine calculates the recommendation.

LLM receives structured information and explains:
- why the event is recommended
- what blocks promotion
- what to do next
- what happens if career goal changes

## Backend idea

Use Go.

Possible endpoints:

GET /employees/{id}
GET /employees/{id}/career-path
GET /employees/{id}/skill-gaps
GET /employees/{id}/recommendations
PUT /employees/{id}/career-goal
GET /events/{id}
POST /navigator/chat

For the hackathon, load JSON dataset in memory first instead of
spending time migrating everything to PostgreSQL.