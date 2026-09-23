package events

import (
	"fmt"
	"sort"

	"careerquest/internal/dataset"
	"careerquest/internal/domain"
)

type Service struct {
	store *dataset.Store
}

func New(store *dataset.Store) *Service { return &Service{store: store} }

type ActivityView struct {
	RecordID      string  `json:"record_id"`
	EventID       string  `json:"event_id"`
	Title         string  `json:"title"`
	EventType     string  `json:"event_type"`
	Format        string  `json:"format"`
	DurationHours float64 `json:"duration_hours"`
	Mandatory     bool    `json:"mandatory"`
	LearningLink  string  `json:"learning_link,omitempty"`
	Date          string  `json:"date"`
	DueDate       string  `json:"due_date,omitempty"`
	Status        string  `json:"status"`
	Category      string  `json:"category"`
	CompletionPct int     `json:"completion_pct"`
	Score         *int    `json:"score,omitempty"`
	AssignedBy    string  `json:"assigned_by"`
}

func (s *Service) EmployeeActivities(employeeID string) ([]ActivityView, error) {
	if _, ok := s.store.Employee(employeeID); !ok {
		return nil, fmt.Errorf("employee %q not found", employeeID)
	}
	activities := s.store.ActivitiesForEmployee(employeeID)
	result := make([]ActivityView, 0, len(activities))
	for _, activity := range activities {
		event, ok := s.store.Event(activity.EventID)
		if !ok {
			continue
		}
		result = append(result, ActivityView{
			RecordID:      activity.RecordID,
			EventID:       event.ID,
			Title:         event.Title,
			EventType:     event.Type,
			Format:        event.Format,
			DurationHours: event.DurationHours,
			Mandatory:     event.Mandatory,
			LearningLink:  event.LearningLink,
			Date:          activity.Date,
			DueDate:       activity.DueDate,
			Status:        activity.Status,
			Category:      activityCategory(activity, event),
			CompletionPct: activity.CompletionPct,
			Score:         activity.Score,
			AssignedBy:    activity.AssignedBy,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date == result[j].Date {
			return result[i].RecordID > result[j].RecordID
		}
		return result[i].Date > result[j].Date
	})
	return result, nil
}

func activityCategory(activity domain.Activity, event domain.Event) string {
	if event.Mandatory {
		return "mandatory"
	}
	switch activity.Status {
	case "completed":
		return "completed"
	case "in_progress":
		return "in_progress"
	case "planned", "enrolled":
		return "planned"
	default:
		return "other"
	}
}

type Analytics struct {
	EventID            string         `json:"event_id"`
	Title              string         `json:"title"`
	TotalRecords       int            `json:"total_records"`
	UniqueParticipants int            `json:"unique_participants"`
	AverageCompletion  float64        `json:"average_completion_pct"`
	StatusCounts       map[string]int `json:"status_counts"`
}

func (s *Service) Analytics(eventID string) (Analytics, error) {
	event, ok := s.store.Event(eventID)
	if !ok {
		return Analytics{}, fmt.Errorf("event %q not found", eventID)
	}
	result := Analytics{EventID: event.ID, Title: event.Title, StatusCounts: make(map[string]int)}
	participants := make(map[string]bool)
	var completionTotal int
	for _, employee := range s.store.Employees() {
		for _, activity := range s.store.ActivitiesForEmployee(employee.ID) {
			if activity.EventID != eventID {
				continue
			}
			result.TotalRecords++
			result.StatusCounts[activity.Status]++
			participants[employee.ID] = true
			completionTotal += activity.CompletionPct
		}
	}
	result.UniqueParticipants = len(participants)
	if result.TotalRecords > 0 {
		result.AverageCompletion = float64(completionTotal) / float64(result.TotalRecords)
	}
	return result, nil
}
