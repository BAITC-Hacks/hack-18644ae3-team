package recommendation

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"careerquest/internal/career"
	"careerquest/internal/domain"
)

const planDepth = 3
const planBeamWidth = 64

type PlanStep struct {
	EventID               string        `json:"event_id"`
	Title                 string        `json:"title"`
	Format                string        `json:"format"`
	DurationHours         float64       `json:"duration_hours"`
	LearningLink          string        `json:"learning_link,omitempty"`
	SkillsDeveloped       []SkillImpact `json:"skills_developed"`
	ReadinessAfterPercent *float64      `json:"readiness_after_percent,omitempty"`
	Explanation           string        `json:"explanation"`
}

type LearningPlan struct {
	Steps                  []PlanStep `json:"steps"`
	TotalDurationHours     float64    `json:"total_duration_hours"`
	ReadinessBeforePercent *float64   `json:"readiness_before_percent,omitempty"`
	ReadinessAfterPercent  *float64   `json:"readiness_after_percent,omitempty"`
	RemainingGapCount      int        `json:"remaining_gap_count"`
	Explanation            string     `json:"explanation"`
}

type planState struct {
	events        []int
	levels        map[string]int
	hours         float64
	readiness     float64
	earlyProgress float64
	afterDate     string
}

// learningPlan searches short sequences, simulating skill gains before checking
// the next course's prerequisites. A bounded beam keeps work small as the
// catalog grows; the result is a suggested path, not a guaranteed optimum.
// events have already passed role, grade, history and availability checks.
func (s *Service) learningPlan(assessment domain.Assessment, profile domain.RoleProfile, events []domain.Event) *LearningPlan {
	if len(assessment.Gaps) == 0 {
		return nil
	}
	sort.Slice(events, func(i, j int) bool { return events[i].ID < events[j].ID })
	initial := planState{levels: assessment.EffectiveSkills, readiness: planReadiness(profile, assessment.EffectiveSkills), afterDate: s.store.Meta().AsOfDate}
	best := initial
	frontier := []planState{initial}
	for depth := 0; depth < planDepth; depth++ {
		next := make([]planState, 0)
		for _, state := range frontier {
			for index, event := range events {
				if slices.Contains(state.events, index) || !meetsPrerequisites(state.levels, event.Prerequisites) {
					continue
				}
				afterDate, available := planNextDate(event, state.afterDate)
				if !available {
					continue
				}
				levels := career.ProjectEvent(state.levels, event)
				improved := false
				for id, level := range levels {
					if level > state.levels[id] {
						improved = true
						break
					}
				}
				if !improved {
					continue
				}
				path := append(slices.Clone(state.events), index)
				readiness := planReadiness(profile, levels)
				candidate := planState{events: path, levels: levels, hours: state.hours + event.DurationHours, readiness: readiness, earlyProgress: state.earlyProgress + readiness, afterDate: afterDate}
				if betterPlan(candidate, best) {
					best = candidate
				}
				next = append(next, candidate)
			}
		}
		sort.Slice(next, func(i, j int) bool { return betterPlan(next[i], next[j]) })
		if len(next) > planBeamWidth {
			next = next[:planBeamWidth]
		}
		frontier = next
	}
	if best.readiness <= initial.readiness {
		return nil
	}
	plan := &LearningPlan{Steps: make([]PlanStep, 0, len(best.events)), TotalDurationHours: best.hours}
	before := assessment.EffectiveSkills
	for position, index := range best.events {
		event := events[index]
		after := career.ProjectEvent(before, event)
		step := PlanStep{EventID: event.ID, Title: event.Title, Format: event.Format, DurationHours: event.DurationHours, LearningLink: event.LearningLink, SkillsDeveloped: []SkillImpact{}}
		changes := make([]string, 0)
		for id, level := range after {
			if level <= before[id] {
				continue
			}
			skill, _ := s.store.Skill(id)
			name := skill.Name
			if name == "" {
				name = id
			}
			step.SkillsDeveloped = append(step.SkillsDeveloped, SkillImpact{SkillID: id, SkillName: name, BeforeLevel: before[id], ProjectedLevel: level, RequiredLevel: profile.RequiredSkills[id], Critical: slices.Contains(profile.CriticalSkills, id)})
		}
		sort.Slice(step.SkillsDeveloped, func(i, j int) bool { return step.SkillsDeveloped[i].SkillID < step.SkillsDeveloped[j].SkillID })
		for _, skill := range step.SkillsDeveloped {
			changes = append(changes, fmt.Sprintf("%s %d to %d", skill.SkillName, skill.BeforeLevel, skill.ProjectedLevel))
		}
		step.Explanation = "Projected skill gains: " + strings.Join(changes, "; ") + "."
		for _, laterIndex := range best.events[position+1:] {
			later := events[laterIndex]
			if !meetsPrerequisites(before, later.Prerequisites) && meetsPrerequisites(after, later.Prerequisites) {
				step.Explanation += fmt.Sprintf(" This unlocks the prerequisites for %s.", later.Title)
			}
		}
		if assessment.Basis == "career_goal" {
			value := round1(career.Readiness(profile, after))
			step.ReadinessAfterPercent = &value
		}
		plan.Steps = append(plan.Steps, step)
		before = after
	}
	for id, required := range profile.RequiredSkills {
		if best.levels[id] < required {
			plan.RemainingGapCount++
		}
	}
	plan.Explanation = fmt.Sprintf("A suggested learning order for %s %s, prioritizing skill progress with extra weight on critical requirements. Equally useful paths favor fewer study hours. Complete and pass each step before starting the next; confirm session dates before enrolling.", assessment.TargetRole, assessment.TargetGrade)
	if assessment.Basis == "career_goal" {
		before, after := round1(initial.readiness), round1(best.readiness)
		plan.ReadinessBeforePercent, plan.ReadinessAfterPercent = &before, &after
	}
	return plan
}

func betterPlan(left, right planState) bool {
	if left.readiness != right.readiness {
		return left.readiness > right.readiness
	}
	if left.hours != right.hours {
		return left.hours < right.hours
	}
	if len(left.events) != len(right.events) {
		return len(left.events) < len(right.events)
	}
	if left.earlyProgress != right.earlyProgress {
		return left.earlyProgress > right.earlyProgress
	}
	return slices.Compare(left.events, right.events) < 0
}

func planReadiness(profile domain.RoleProfile, levels map[string]int) float64 {
	// Ignore floating-point accumulation differences from map iteration order.
	return math.Round(career.Readiness(profile, levels)*1e6) / 1e6
}

func planNextDate(event domain.Event, afterDate string) (string, bool) {
	if event.Format == "self_paced" {
		return afterDate, true
	}
	upcoming, available := availability(event, afterDate)
	if !available {
		return "", false
	}
	date, err := time.Parse("2006-01-02", slices.Min(upcoming))
	if err != nil {
		return "", false
	}
	// A subsequent course cannot start before its prerequisite session.
	return date.AddDate(0, 0, 1).Format("2006-01-02"), true
}
