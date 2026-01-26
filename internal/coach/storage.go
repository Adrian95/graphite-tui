package coach

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SaveState persists coach progress to ~/.graphite-coach.json
func SaveState(state CoachState) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path := filepath.Join(homeDir, ".graphite-coach.json")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadState retrieves saved coach progress
func LoadState() CoachState {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return defaultCoachState()
	}

	path := filepath.Join(homeDir, ".graphite-coach.json")
	data, err := os.ReadFile(path)
	if err != nil {
		// First time - create default state
		return defaultCoachState()
	}

	var state CoachState
	if err := json.Unmarshal(data, &state); err != nil {
		return defaultCoachState()
	}

	// Ensure all lessons exist (in case new ones were added)
	for id, lesson := range initializeLessons() {
		if _, exists := state.Lessons[id]; !exists {
			state.Lessons[id] = lesson
		}
	}

	return state
}

// defaultCoachState creates initial coach state with all lessons
func defaultCoachState() CoachState {
	return CoachState{
		Enabled: true, // Default ON
		Lessons: initializeLessons(),
	}
}

// initializeLessons creates default lesson tracking
func initializeLessons() map[string]*Lesson {
	lessons := []struct {
		id    string
		topic Topic
	}{
		{"first_commit", TopicStacking},
		{"second_commit", TopicStacking},
		{"stack_visualization", TopicStacking},
		{"first_sync", TopicSyncing},
		{"conflict_intro", TopicConflicts},
		{"amending", TopicAmending},
		{"iterate", TopicAmending},
		{"first_submit", TopicSubmitting},
		{"multi_pr", TopicSubmitting},
		{"merge_cleanup", TopicMerging},
	}

	m := make(map[string]*Lesson)
	for _, l := range lessons {
		m[l.id] = &Lesson{
			ID:      l.id,
			Topic:   l.topic,
			Learned: false,
		}
	}
	return m
}
