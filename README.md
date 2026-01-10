# Graphite TUI

```
   ┌─────────────────────────────────────┐
   │  GRAPHITE TUI          ┌──────────┐ │
   │  v1.9.0                │  SPEED   │ │
   │                        │ ❯ Ship   │ │
   │  ┌───────────────────┐ │   Iterate│ │
   │  │  feat/auth-flow   │ │   Reset  │ │
   │  │  ↑2 ahead         │ │   Undo   │ │
   │  └───────────────────┘ └──────────┘ │
   │                                     │
   │  CHANGES (3)                        │
   │  M src/auth.ts                      │
   │  + src/login.tsx                    │
   │  ? .env.local                       │
   │                                     │
   │  → Amends commit → pushes → updates │
   │───────────────────────────────────  │
   │ s New  p PR  y Sync  d Merge  ? Help│
   └─────────────────────────────────────┘
```

**The keyboard-first TUI for [Graphite](https://graphite.dev) power users.**

Built for developers who live in the terminal and want to ship fast without breaking flow. No more context-switching to browsers. No more typing `gt` commands. Just vibes.

## Why?

You're deep in flow. Code is flowing. You want to:
- Commit and push without thinking about branch names
- Update your PR with one keystroke
- See your stack without leaving the terminal
- Never type `gt create`, `gt modify`, `gt submit` again

This is that.

## Install

```bash
go install github.com/Adrian95/graphite-tui/cmd/graphite-tui@latest
```

Then run it:

```bash
graphite-tui
```

> Requires [Graphite CLI](https://graphite.dev/docs/cli/installation) (`gt`) to be installed and authenticated.

## The SPEED Box

The SPEED box is your command center. Navigate with `↑↓`, execute with `Enter`.

| Action | What it does |
|--------|--------------|
| **Ship** | *Context-aware.* On main with changes? Creates branch + commits + submits PR. On a branch? Amends + pushes. No changes but ahead? Submits PR. It just knows. |
| **Iterate** | Amend your last commit and push. Your PR updates automatically. One keystroke to update reviewers. |
| **Reset** | Nuke everything. Hard reset to last commit. For when you need a fresh start. |
| **Undo** | Reverse your last Graphite operation. Made a mistake? Ctrl+Z for git. |

## Workflow Keys

| Key | Action | Description |
|-----|--------|-------------|
| `s` | **Start** | The commit wizard. Walks you through type → scope → message → commit. Creates a branch automatically. |
| `p` | **Share** | Submit your PR for review. Triggers preview builds. |
| `y` | **Sync** | Pull latest from your team. Stay up to date. |
| `d` | **Done** | Merge your approved PR and clean up. One key to rule them all. |

## Navigation

| Key | Action | Description |
|-----|--------|-------------|
| `g` | **Stack** | Visual stack map. See your entire branch hierarchy. Navigate and checkout. |
| `x` | **Rescue** | Ghost branch recovery. Fix orphaned branches after merges. |
| `Tab` | **Expand** | Toggle full file list. See all your changes. |
| `h` | **Hooks** | Toggle pre-commit hooks on/off. Skip the linter when you're iterating fast. |
| `u` | **Update** | Check for updates. Stay fresh. |
| `?` | **Help** | Show all keybindings. |
| `q` | **Quit** | Exit. |

## The Commit Wizard

Press `s` to start the guided commit flow:

```
Step 1 of 4: What kind of change is this?
❯ feat       New feature        "add dark mode"
  fix        Bug fix            "resolve login crash"
  refactor   Code improvement   "extract auth logic"
  docs       Documentation      "update API docs"
  ...

Step 2 of 4: What part of the codebase? (Optional)
Component name: auth, navbar, api, etc.

Step 3 of 4: Describe the change
Use present tense: 'add button' not 'added button'

Step 4 of 4: Ready to commit?
┌────────────────────────────────┐
│ feat(auth): add OAuth support  │
└────────────────────────────────┘
```

## Smart Ship

The `Ship` command adapts to your context:

| Context | Ship does this |
|---------|----------------|
| On `main` with changes | Creates branch → commits (wizard) → submits PR |
| On branch with changes | Amends commit → pushes → updates PR |
| On branch, no changes, ahead of remote | Submits PR |
| Clean, nothing to do | Tells you "Nothing to ship" |

One command. Always the right action.

## Stack Visualization

Press `g` to see your branch stack:

```
Stack Map

  ○ main
    ○ feat/auth-base
      ● feat/auth-oauth  ← you are here
        ○ feat/auth-tests

● you  ○ branch  ⦿ selected

[↑↓] Navigate  [Enter] Checkout  [Esc] Back
```

Navigate your stack visually. Checkout any branch with `Enter`.

## Design

- **Dark mode first** — Designed for modern terminals (Ghostty, Wezterm, Kitty, iTerm2)
- **High contrast** — Easy on the eyes during long sessions
- **Minimal chrome** — Just the information you need
- **Instant feedback** — See command output in real-time

## Requirements

- [Go 1.21+](https://go.dev/dl/)
- [Graphite CLI](https://graphite.dev/docs/cli/installation) installed and authenticated
- A terminal that supports modern rendering (most do)

## Build from Source

```bash
git clone https://github.com/Adrian95/graphite-tui.git
cd graphite-tui
go install ./cmd/graphite-tui
```

## Philosophy

This tool exists because the best workflow is the one you don't think about.

Graphite's CLI is powerful but verbose. You shouldn't need to remember if it's `gt create`, `gt commit`, or `gt modify`. You shouldn't need to type branch names. You shouldn't need to leave your terminal.

Just `Ship`.

---

**Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).**
