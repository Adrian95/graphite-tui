package coach

import (
	"fmt"
	"strings"

	"github.com/Adrian95/graphite-tui/v2/internal/git"
	"github.com/Adrian95/graphite-tui/v2/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// --- Stacking Lessons ---

// FirstCommitExplainer teaches the basics of creating a commit with Graphite
func FirstCommitExplainer(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
	branchName := local.Branch
	if branchName == "" {
		branchName = "(new branch)"
	}

	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true)

	return ExplainerData{
		Title: "🎉 You Created Your First Commit!",
		WhatHappened: fmt.Sprintf(
			"Graphite created a new branch and committed your changes.\n\n"+
				"Branch: %s\n"+
				"Your work is now saved as a checkpoint you can return to.",
			accentStyle.Render(branchName),
		),
		WhyItMatters: "Unlike regular git commits, Graphite creates a BRANCH for each commit. " +
			"This lets you submit each piece of work as a separate PR while building on top of it.\n\n" +
			"Each commit = One focused change = One easy-to-review PR",
		RepoState:    renderStackVisualization(stack, local.Branch),
		NextSteps: []string{
			"Keep working and press [s] to create ANOTHER commit on top",
			"Or press [p] to submit this as a Pull Request",
			"Press [y] to sync with main and stay up-to-date",
		},
		LessonID: "first_commit",
		CanSkip:  true,
	}
}

// SecondCommitExplainer teaches about stacking commits
func SecondCommitExplainer(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
	previousBranch := getPreviousBranch(stack, local.Branch)
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true)

	return ExplainerData{
		Title: "🚀 You're Stacking!",
		WhatHappened: fmt.Sprintf(
			"You just created a SECOND commit on top of your first one.\n\n"+
				"Graphite created branch: %s\n"+
				"Built on top of: %s",
			accentStyle.Render(local.Branch),
			accentStyle.Render(previousBranch),
		),
		WhyItMatters: "This is the POWER of stacking! You can:\n\n" +
			"  • Keep working without waiting for reviews\n" +
			"  • Submit small, focused PRs that are easy to review\n" +
			"  • Change the bottom PR without affecting the top\n\n" +
			"Each PR builds on the previous one, but they review independently.",
		RepoState: renderStackWithHighlight(stack, local.Branch),
		NextSteps: []string{
			"Submit BOTH PRs with [p] - they'll be linked automatically",
			"Continue stacking with [s] to create a third commit",
			"View your full stack anytime with [g]",
		},
		LessonID: "second_commit",
		CanSkip:  true,
	}
}

// StackVisualizationExplainer explains the stack view
func StackVisualizationExplainer(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
	stackSize := len(stack.Stack)

	return ExplainerData{
		Title: "📚 Your Stack Visualization",
		WhatHappened: fmt.Sprintf(
			"You opened the stack view ([g]).\n\n"+
				"This shows all %d branches in your stack, from bottom (main) to top (current work).",
			stackSize,
		),
		WhyItMatters: "The stack view helps you:\n\n" +
			"  • See the entire dependency chain\n" +
			"  • Quickly jump between branches with [Enter]\n" +
			"  • Understand what builds on what\n\n" +
			"The * indicates your current branch.",
		RepoState: renderStackWithHighlight(stack, local.Branch),
		NextSteps: []string{
			"Press [Enter] to checkout a different branch in the stack",
			"Use [↑] and [↓] to navigate",
			"Press [Esc] to return to dashboard",
		},
		LessonID: "stack_visualization",
		CanSkip:  true,
	}
}

// --- Syncing Lessons ---

// FirstSyncExplainer teaches about syncing with main
func FirstSyncExplainer(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
	return ExplainerData{
		Title: "🔄 You Synced Your Stack!",
		WhatHappened: "Graphite pulled the latest changes from main and rebased your stack on top.\n\n" +
			"Your branches now include all the latest work from your team.",
		WhyItMatters: "Syncing regularly prevents conflicts and keeps your work up-to-date.\n\n" +
			"Graphite rebases your ENTIRE STACK automatically, so all your PRs stay current.\n\n" +
			"Do this daily to stay in sync with your team!",
		RepoState: renderStackVisualization(stack, local.Branch),
		NextSteps: []string{
			"Make this a habit - sync before starting new work",
			"If conflicts appear, Graphite will help you resolve them",
			"After syncing, your PRs automatically update on GitHub",
		},
		LessonID: "first_sync",
		CanSkip:  true,
	}
}

// ConflictIntroExplainer explains merge conflicts
func ConflictIntroExplainer(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
	return ExplainerData{
		Title: "⚠️  Merge Conflicts Detected",
		WhatHappened: "Your changes conflicted with updates from main.\n\n" +
			"This happens when you and someone else edited the same lines of code.",
		WhyItMatters: "Conflicts are normal! They just mean Git needs YOUR help to decide which changes to keep.\n\n" +
			"Graphite will guide you through resolving conflicts one branch at a time.",
		NextSteps: []string{
			"Resolve conflicts in your editor",
			"Stage the fixed files",
			"Run 'gt continue' to finish the rebase",
		},
		LessonID: "conflict_intro",
		CanSkip:  true,
	}
}

// --- Amending Lessons ---

// AmendingExplainer teaches about modifying existing commits
func AmendingExplainer(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
	return ExplainerData{
		Title: "✏️  You Amended a Commit!",
		WhatHappened: "Instead of creating a NEW commit, you modified your LAST commit.\n\n" +
			"Your latest changes were added to the existing commit.",
		WhyItMatters: "Amending lets you fix mistakes or add forgotten changes without creating tiny " +
			"'oops' commits.\n\n" +
			"Perfect for:\n" +
			"  • Fixing typos\n" +
			"  • Adding forgotten files\n" +
			"  • Improving code based on review feedback",
		RepoState: renderStackVisualization(stack, local.Branch),
		NextSteps: []string{
			"Press [f] to push your changes and update the PR",
			"Reviewers will see your updates automatically",
			"Keep amending until the PR is perfect!",
		},
		LessonID: "amending",
		CanSkip:  true,
	}
}

// IterateExplainer teaches the iterate workflow (amend + push)
func IterateExplainer(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
	return ExplainerData{
		Title: "🔁 You Iterated on Your PR!",
		WhatHappened: "Graphite amended your commit AND pushed it to GitHub.\n\n" +
			"Your Pull Request is now updated with your latest changes.",
		WhyItMatters: "This is the FAST way to respond to code review feedback:\n\n" +
			"  1. Make changes locally\n" +
			"  2. Press [f] to amend + push\n" +
			"  3. Reviewers see updates instantly\n\n" +
			"No separate 'fix review comments' commits needed!",
		RepoState: renderStackVisualization(stack, local.Branch),
		NextSteps: []string{
			"Repeat this workflow for each round of feedback",
			"When approved, press [d] to merge and clean up",
			"Your PR history stays clean with no 'fix typo' commits",
		},
		LessonID: "iterate",
		CanSkip:  true,
	}
}

// --- Submitting Lessons ---

// FirstSubmitExplainer teaches about creating PRs
func FirstSubmitExplainer(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
	return ExplainerData{
		Title: "🚢 You Submitted Your First PR!",
		WhatHappened: "Graphite pushed your branch to GitHub and created a Pull Request.\n\n" +
			"Your code is now ready for review!",
		WhyItMatters: "PRs are how your team reviews code before it goes to production.\n\n" +
			"Graphite makes PRs easy:\n" +
			"  • Automatically creates them from your branches\n" +
			"  • Links stacked PRs together\n" +
			"  • Updates them when you make changes",
		NextSteps: []string{
			"Share the PR link with your team for review",
			"Make changes based on feedback, then press [f] to update",
			"When approved, press [d] to merge and clean up",
		},
		LessonID: "first_submit",
		CanSkip:  true,
	}
}

// MultiPRExplainer teaches about submitting stacked PRs
func MultiPRExplainer(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
	stackSize := len(stack.Stack)

	return ExplainerData{
		Title: "🎯 You Submitted Multiple Stacked PRs!",
		WhatHappened: fmt.Sprintf(
			"Graphite submitted %d Pull Requests at once!\n\n"+
				"Each PR is linked to show the dependency chain.",
			stackSize,
		),
		WhyItMatters: "Stacked PRs let reviewers focus on ONE change at a time:\n\n" +
			"  • Each PR is small and focused\n" +
			"  • They can review them independently\n" +
			"  • Merge them one-by-one as they're approved\n\n" +
			"This is MUCH easier to review than one giant PR!",
		RepoState: renderStackWithHighlight(stack, local.Branch),
		NextSteps: []string{
			"Reviewers can approve PRs in any order",
			"As bottom PRs merge, top ones automatically update",
			"Use [g] to see your stack and jump between PRs",
		},
		LessonID: "multi_pr",
		CanSkip:  true,
	}
}

// --- Merging Lessons ---

// MergeCleanupExplainer teaches about the merge + cleanup workflow
func MergeCleanupExplainer(local git.LocalStatusMsg, stack git.StackStatusMsg) ExplainerData {
	return ExplainerData{
		Title: "✨ PR Merged & Cleaned Up!",
		WhatHappened: "Graphite merged your approved PR and cleaned up the branch.\n\n" +
			"Your code is now in production!",
		WhyItMatters: "The merge + cleanup workflow keeps your repo tidy:\n\n" +
			"  • PR gets merged to main\n" +
			"  • Local branch is deleted\n" +
			"  • Your stack automatically updates\n\n" +
			"No manual cleanup needed!",
		RepoState: renderStackVisualization(stack, local.Branch),
		NextSteps: []string{
			"Continue working on your next commit in the stack",
			"Or press [y] to sync and get latest changes",
			"Your remaining PRs are automatically updated",
		},
		LessonID: "merge_cleanup",
		CanSkip:  true,
	}
}

// --- Helper Functions ---

// renderStackVisualization creates a simple stack visualization
func renderStackVisualization(stack git.StackStatusMsg, currentBranch string) string {
	if len(stack.Stack) == 0 {
		return ui.SubtitleStyle.Render("(No stack to display)")
	}

	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true)

	var lines []string
	for i := len(stack.Stack) - 1; i >= 0; i-- {
		branch := stack.Stack[i]
		isCurrent := branch == currentBranch

		prefix := "  "
		if isCurrent {
			prefix = accentStyle.Render("* ")
		}

		branchStyle := lipgloss.NewStyle()
		if isCurrent {
			branchStyle = branchStyle.Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true)
		}

		lines = append(lines, fmt.Sprintf("%s%s", prefix, branchStyle.Render(branch)))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ui.ColorBorder)).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}

// renderStackWithHighlight creates a stack visualization with the current branch highlighted
func renderStackWithHighlight(stack git.StackStatusMsg, currentBranch string) string {
	return renderStackVisualization(stack, currentBranch)
}

// getPreviousBranch returns the branch that currentBranch is built on top of
func getPreviousBranch(stack git.StackStatusMsg, currentBranch string) string {
	for i, branch := range stack.Stack {
		if branch == currentBranch && i > 0 {
			return stack.Stack[i-1]
		}
	}
	return "main"
}
