package coach

import "time"

// Topic represents a coaching area
type Topic string

const (
	TopicStacking   Topic = "stacking"   // Branch-on-branch workflows
	TopicSyncing    Topic = "syncing"    // Rebasing, pulling updates
	TopicAmending   Topic = "amending"   // Modifying commits
	TopicSubmitting Topic = "submitting" // Creating PRs
	TopicMerging    Topic = "merging"    // Cleanup & landing
	TopicConflicts  Topic = "conflicts"  // Handling merge conflicts
)

// Lesson represents what was learned
type Lesson struct {
	ID         string    `json:"id"`          // Unique identifier
	Topic      Topic     `json:"topic"`       // Which area this teaches
	Learned    bool      `json:"learned"`     // User has seen this
	TimesShown int       `json:"times_shown"` // Track repetition
	LastShown  time.Time `json:"last_shown"`  // When last displayed
}

// CoachState tracks user progress
type CoachState struct {
	Enabled      bool               `json:"enabled"`       // Coach mode on/off
	Lessons      map[string]*Lesson `json:"lessons"`       // Lesson tracking
	CurrentTopic Topic              `json:"current_topic"` // What we're currently teaching
}

// ExplainerData shown after operations
type ExplainerData struct {
	Title        string   // "You just stacked a commit!"
	WhatHappened string   // Plain English explanation
	WhyItMatters string   // Why this is useful
	RepoState    string   // Current state visualization
	NextSteps    []string // Suggested actions
	LessonID     string   // For tracking
	CanSkip      bool     // Show "don't show again"
}
