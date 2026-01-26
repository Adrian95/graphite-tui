package coach

import (
	"strings"

	"github.com/Adrian95/graphite-tui/v2/internal/git"
)

// Trigger detects when to show coaching
type Trigger struct {
	Command        string                                                               // What command just ran
	ShouldExplain  func(CoachState, git.LocalStatusMsg, git.StackStatusMsg) bool        // Check if we should show
	BuildExplainer func(git.LocalStatusMsg, git.StackStatusMsg) ExplainerData           // Create the explainer
}

// All triggers for different commands
var triggers = []Trigger{
	{
		Command: "gt create",
		ShouldExplain: func(cs CoachState, local git.LocalStatusMsg, stack git.StackStatusMsg) bool {
			if !cs.Enabled {
				return false
			}

			// Check if this is the first or second commit
			stackSize := len(stack.Stack)

			// First commit lesson
			if lesson, ok := cs.Lessons["first_commit"]; ok && !lesson.Learned && stackSize <= 1 {
				return true
			}

			// Second commit lesson (stacking)
			if lesson, ok := cs.Lessons["second_commit"]; ok && !lesson.Learned && stackSize == 2 {
				return true
			}

			return false
		},
		BuildExplainer: func(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
			stackSize := len(stack.Stack)
			if stackSize <= 1 {
				return FirstCommitExplainer(local, stack)
			}
			return SecondCommitExplainer(local, stack)
		},
	},
	{
		Command: "turbo ship",
		ShouldExplain: func(cs CoachState, local git.LocalStatusMsg, stack git.StackStatusMsg) bool {
			if !cs.Enabled {
				return false
			}

			stackSize := len(stack.Stack)

			// First commit lesson (turbo ship also creates commits)
			if lesson, ok := cs.Lessons["first_commit"]; ok && !lesson.Learned && stackSize <= 1 {
				return true
			}

			// Second commit lesson
			if lesson, ok := cs.Lessons["second_commit"]; ok && !lesson.Learned && stackSize == 2 {
				return true
			}

			return false
		},
		BuildExplainer: func(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
			stackSize := len(stack.Stack)
			if stackSize <= 1 {
				return FirstCommitExplainer(local, stack)
			}
			return SecondCommitExplainer(local, stack)
		},
	},
	{
		Command: "gt modify",
		ShouldExplain: func(cs CoachState, local git.LocalStatusMsg, stack git.StackStatusMsg) bool {
			if !cs.Enabled {
				return false
			}
			lesson, ok := cs.Lessons["amending"]
			return ok && !lesson.Learned
		},
		BuildExplainer: AmendingExplainer,
	},
	{
		Command: "iterate",
		ShouldExplain: func(cs CoachState, local git.LocalStatusMsg, stack git.StackStatusMsg) bool {
			if !cs.Enabled {
				return false
			}
			lesson, ok := cs.Lessons["iterate"]
			return ok && !lesson.Learned
		},
		BuildExplainer: IterateExplainer,
	},
	{
		Command: "gt submit",
		ShouldExplain: func(cs CoachState, local git.LocalStatusMsg, stack git.StackStatusMsg) bool {
			if !cs.Enabled {
				return false
			}

			stackSize := len(stack.Stack)

			// First submit
			if lesson, ok := cs.Lessons["first_submit"]; ok && !lesson.Learned && stackSize <= 1 {
				return true
			}

			// Multi-PR submit
			if lesson, ok := cs.Lessons["multi_pr"]; ok && !lesson.Learned && stackSize > 1 {
				return true
			}

			return false
		},
		BuildExplainer: func(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
			stackSize := len(stack.Stack)
			if stackSize <= 1 {
				return FirstSubmitExplainer(local, stack)
			}
			return MultiPRExplainer(local, stack)
		},
	},
	{
		Command: "gt sync",
		ShouldExplain: func(cs CoachState, local git.LocalStatusMsg, stack git.StackStatusMsg) bool {
			if !cs.Enabled {
				return false
			}
			lesson, ok := cs.Lessons["first_sync"]
			return ok && !lesson.Learned
		},
		BuildExplainer: FirstSyncExplainer,
	},
	{
		Command: "merge",
		ShouldExplain: func(cs CoachState, local git.LocalStatusMsg, stack git.StackStatusMsg) bool {
			if !cs.Enabled {
				return false
			}
			lesson, ok := cs.Lessons["merge_cleanup"]
			return ok && !lesson.Learned
		},
		BuildExplainer: MergeCleanupExplainer,
	},
}

// CheckTriggers runs after command completion and returns explainer if needed
func CheckTriggers(coachState *CoachState, lastCommand string, local git.LocalStatusMsg, stack git.StackStatusMsg) *ExplainerData {
	if !coachState.Enabled {
		return nil
	}

	for _, trigger := range triggers {
		if strings.Contains(lastCommand, trigger.Command) {
			if trigger.ShouldExplain(*coachState, local, stack) {
				explainer := trigger.BuildExplainer(local, stack)
				return &explainer
			}
		}
	}

	return nil
}
