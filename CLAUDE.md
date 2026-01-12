# CLAUDE.md - AI Assistant Guide for Graphite TUI

This document provides comprehensive guidance for AI assistants working on the Graphite TUI codebase.

## Project Overview

**Graphite TUI** is a keyboard-first Terminal User Interface for [Graphite CLI](https://graphite.dev) power users. It eliminates repetitive `gt` commands by providing intuitive keybindings for common stacked PR workflows. Version is embedded at build time from git tags (see "Release & Update Architecture" section).

**Tech Stack:**
- **Language:** Go 1.21+
- **TUI Framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm-inspired MVU architecture)
- **Styling:** [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Components:** [Bubbles](https://github.com/charmbracelet/bubbles) (spinner, textinput, viewport)

## Directory Structure

```
graphite-tui/
├── main.go                     # Application entry point & TEA model
├── go.mod / go.sum             # Go module dependencies
├── README.md                   # User-facing documentation
├── internal/
│   ├── config/
│   │   └── version.go          # Version management & auto-update
│   ├── git/
│   │   ├── commands.go         # Git/Graphite command definitions
│   │   └── executor.go         # Shell command execution engine
│   ├── state/
│   │   ├── state.go            # State machine interface
│   │   └── states.go           # Concrete state implementations
│   └── ui/
│       ├── ui.go               # Theme, colors, base styles
│       ├── watcher.go          # Periodic status refresh
│       └── views/
│           ├── context.go      # Render context & layout helpers
│           ├── dashboard.go    # Main dashboard view
│           ├── sidebar.go      # Sidebar with SPEED box
│           ├── dialogs.go      # Modal dialogs
│           └── wizard.go       # 4-step commit wizard
```

## Architecture

### Bubble Tea MVU Pattern

This project follows the **Model-View-Update** (MVU) architecture from [Bubble Tea](https://pkg.go.dev/github.com/charmbracelet/bubbletea):

```go
// Model: Application state
type model struct {
    stateID       state.StateID
    dashboardData views.DashboardViewData
    // ... other fields
}

// Init: Returns initial commands (timers, async operations)
func (m model) Init() tea.Cmd { ... }

// Update: Handles messages and returns new state + commands
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }

// View: Renders UI based on current state
func (m model) View() string { ... }
```

**Key Patterns:**
- Return `tea.Cmd` for async operations (never block in Update)
- Use `tea.Batch()` to combine multiple commands
- All state changes happen through `Update()`
- `View()` is a pure function of the model

### State Machine

The application uses a finite state machine (`internal/state/`) with 15 states:

| State | Purpose |
|-------|---------|
| `Dashboard` | Main view with branch info and SPEED box |
| `Menu` | Extended menu options |
| `Input` | Text input dialogs |
| `Running` | Command execution in progress |
| `Output` | Display command results |
| `WizardType` | Commit wizard step 1 (type selection) |
| `WizardScope` | Commit wizard step 2 (scope input) |
| `WizardSummary` | Commit wizard step 3 (message input) |
| `WizardPreview` | Commit wizard step 4 (confirmation) |
| `Stack` | Branch stack visualization |
| `Help` | Keybinding reference |
| `Update` | Version check/update |
| `Confirm` | Confirmation dialogs |
| `PostCommit` | After commit actions |
| `Startup` | Initial loading |

**Transition Example:**
```go
// In handleDashboardKeys()
case "s":
    m.stateID = state.WizardType
    m.wizardData = views.WizardViewData{Stage: 1, CommitTypes: git.GetCommitTypes()}
    return m, nil
```

### Command Execution

Commands are executed via `internal/git/executor.go`:

```go
// Non-interactive (background) execution
git.ExecuteAsync(cmd, timeout) tea.Cmd

// Interactive (terminal passthrough)
git.ExecuteInteractive(cmd, commitMsg, skipHooks) tea.Cmd

// Chain multiple commands
git.ExecuteChain(cmds) tea.Cmd
```

**Safety Features:**
- Non-interactive env vars prevent editor popups (`GIT_EDITOR=true`, `GT_NON_INTERACTIVE=1`)
- 30-second default timeout
- Branch name sanitization for special characters

## Code Conventions

### File Organization

```go
// --- Section Name ---   // Use section headers

type Model struct { ... } // Types first

func NewModel() { ... }   // Constructors

func (m Model) Method()   // Methods grouped by purpose
```

### Naming Conventions

- `CamelCase` for exported types/functions
- Descriptive names: `ExecuteTurboShip`, `buildNonInteractiveEnv`
- State IDs as constants: `state.Dashboard`, `state.WizardType`
- Messages end with `Msg`: `StatusMsg`, `CmdFinishedMsg`

### Error Handling

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to execute command: %w", err)
}

// Check specific error types
if errors.Is(err, context.DeadlineExceeded) {
    // Handle timeout
}
```

### Styling (Lipgloss)

All styles are defined in `internal/ui/ui.go`:

```go
// Color palette
const (
    ColorBackground = "#000000"  // Pure black
    ColorText       = "#EDEDED"  // Off-white
    ColorAccent     = "#FF0080"  // Hot pink
    ColorSuccess    = "#50E3C2"  // Teal
    ColorWarning    = "#F5A623"  // Orange
    ColorError      = "#FF3366"  // Red-pink
    ColorBorder     = "#333333"  // Dark gray
)

// Predefined styles
var TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(...)
```

## Key Keybindings (main.go)

### Dashboard
| Key | Handler | Action |
|-----|---------|--------|
| `q` | `handleDashboardKeys` | Quit |
| `s` | `handleDashboardKeys` | Start commit wizard |
| `p` | `handleDashboardKeys` | Submit PR (`gt submit`) |
| `f` | `handleDashboardKeys` | Iterate (amend + push) |
| `y` | `handleDashboardKeys` | Sync (`gt sync`) |
| `d` | `handleDashboardKeys` | Merge + cleanup |
| `g` | `handleDashboardKeys` | Stack visualization |
| `x` | `handleDashboardKeys` | Ghost branch rescue |
| `h` | `handleDashboardKeys` | Toggle hooks |
| `R` | `handleDashboardKeys` | Hard reset |
| `z` | `handleDashboardKeys` | Undo |
| `?` | `handleDashboardKeys` | Help |

### SPEED Box Actions (via Enter)
| Index | Action | Behavior |
|-------|--------|----------|
| 0 | Ship | Context-aware: creates branch/commits/submits OR amends/pushes |
| 1 | Iterate | Amend last commit + push |
| 2 | Reset | Hard reset (`git reset --hard HEAD && git clean -fd`) |
| 3 | Undo | `gt undo` |

## Important Functions

### main.go
- `initialModel()` - Creates initial application state
- `executeSpeedAction()` - Context-aware shipping logic
- `handleDashboardKeys()` - Main view key handling
- `handleWizard*Keys()` - Commit wizard navigation

### internal/git/commands.go
- `CheckGitStatus()` - Returns branch, files, ahead/behind counts
- `LoadStack()` - Parses `gt log short` output
- `ExecuteCommit()` - Creates branch + commits via `gt create`
- `ExecuteTurboShip()` - Full workflow: branch → commit → submit
- `ExecuteIterate()` - Amend + push + update PR

### internal/git/executor.go
- `ExecuteAsync()` - Background command execution
- `ExecuteInteractive()` - Terminal passthrough execution
- `buildNonInteractiveEnv()` - Prevents editor popups

### internal/config/version.go
- `GetBuildInfo()` - Returns version, commit, build time from Go's embedded metadata
- `GetCurrentVersion()` - Returns display version (embedded for installs, fallback+dev for local)
- `CheckForUpdates()` - Queries GitHub releases API for latest tag
- `PerformUpdate()` / `PerformUpdateToVersion()` - Runs `go install` with verification
- `getInstalledBinaryVersion()` - Inspects binary metadata to verify installed version
- `IsNewerVersion()` - Semantic version comparison
- `DiagnoseVersionIssue()` - Debug helper for version mismatch issues

## Graphite CLI Commands Used

The TUI wraps these [Graphite CLI](https://graphite.com/docs/cli-quick-start) commands:

| Command | Purpose |
|---------|---------|
| `gt log short` | Get stack visualization |
| `gt create -a -m "msg"` | Create branch + commit |
| `gt modify -a -m "msg"` | Amend current commit |
| `gt submit` | Create/update PR |
| `gt sync` | Pull latest + cleanup merged |
| `gt merge` | Merge approved PR |
| `gt checkout <branch>` | Switch branches |
| `gt undo` | Reverse last operation |
| `gt init` | Initialize Graphite in repo |

## Development Workflow

### Building
```bash
# Install from source
go install ./...

# Or build binary
go build -o graphite-tui .
```

### Running
```bash
# Must be in a git repository with Graphite initialized
cd your-project
graphite-tui
```

### Adding New Features

1. **New State:** Add to `internal/state/states.go`, update transitions in `state.go`
2. **New View:** Add to `internal/ui/views/`, render in `main.go View()`
3. **New Keybinding:** Add case in appropriate `handle*Keys()` function
4. **New Command:** Add wrapper in `internal/git/commands.go`

### Debugging

```go
// Log to file (can't use stdout - TUI uses it)
tea.LogToFile("debug.log", "debug")
```

## Testing

Currently no test suite. When adding tests:

- Unit test state transitions in `internal/state/`
- Unit test command parsing in `internal/git/`
- Unit test version comparison in `internal/config/`
- Mock `exec.Command` for command execution tests

## Release & Update Architecture

### Version Management

**CRITICAL: Version is embedded by Go, not hardcoded.**

The application version comes from Go's build system, NOT from a hardcoded constant:

```go
// internal/config/version.go

// FallbackVersion is ONLY used for local dev builds
const FallbackVersion = "v0.0.0-dev"

// GetCurrentVersion returns the REAL version from Go's build info
func GetCurrentVersion() string {
    info := GetBuildInfo()
    if info.IsDev {
        return info.Version + " (dev)"
    }
    return info.Version  // e.g., "v1.9.3" from git tag
}
```

When users install via `go install github.com/Adrian95/graphite-tui@v1.9.3`, Go automatically embeds `v1.9.3` into the binary via `debug.ReadBuildInfo()`.

**DO NOT:**
- Manually update version constants for releases
- Use hardcoded versions for comparisons
- Assume `FallbackVersion` represents the current release

### Release Process

**Step-by-step release checklist:**

```bash
# 1. Ensure all changes are committed
git status  # Should be clean

# 2. Create annotated tag (MUST match semver format)
git tag -a v1.9.4 -m "Release v1.9.4: Brief description"

# 3. Push the tag to GitHub
git push origin v1.9.4

# 4. Create GitHub Release
#    - Go to GitHub → Releases → "Draft a new release"
#    - Select the tag you just pushed
#    - Title: "v1.9.4"
#    - Description: Changelog/release notes
#    - Publish release

# 5. Verify the release works
GOPROXY=direct go install github.com/Adrian95/graphite-tui@v1.9.4
graphite-tui  # Check version displays correctly
```

**Common Release Mistakes:**
| Mistake | Symptom | Fix |
|---------|---------|-----|
| Tag not pushed | `go install` gets old version | `git push origin <tag>` |
| Release without tag | GitHub shows release but install fails | Create and push matching tag |
| Tag format wrong | `v1.9.4` works, `1.9.4` may not | Always use `v` prefix |
| Proxy cache | Old version despite correct tag | Use `GOPROXY=direct` |

### Update Mechanism

The update system (`internal/config/version.go`) works as follows:

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Check GitHub   │────▶│ Compare Versions │────▶│  go install     │
│  Releases API   │     │ (semver parse)   │     │  @latest/tag    │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                                          │
                                                          ▼
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ Show result to  │◀────│ Verify installed │◀────│ Copy binary to  │
│    user         │     │    version       │     │ current path    │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

**Key functions:**
- `CheckForUpdates()` - Queries GitHub API for latest release tag
- `PerformUpdate()` - Runs `go install` with `GOPROXY=direct`
- `getInstalledBinaryVersion()` - Inspects new binary to verify version
- `GetBuildInfo()` - Returns embedded version, commit, build time

**Update verification:**
After `go install`, the system verifies the new binary's version matches expected:
```go
installedVersion, _ := getInstalledBinaryVersion(targetBinary)
if normalizeVersion(targetVersion) != normalizeVersion(installedVersion) {
    // Warning: version mismatch detected
}
```

### Build Metadata

The `GetBuildInfo()` function extracts:
- **Version**: From git tag (e.g., `v1.9.3`)
- **Commit**: Short SHA from `vcs.revision`
- **BuildTime**: From `vcs.time`
- **GoVersion**: Go version used to compile
- **IsDev**: True if built locally vs installed

```go
info := config.GetBuildInfo()
// info.Version = "v1.9.3"
// info.Commit = "abc1234"
// info.BuildTime = "2024-01-15T10:30:00Z"
// info.GoVersion = "go1.21.5"
// info.IsDev = false
```

### Troubleshooting Updates

**"Update shows success but version unchanged":**
1. Check GitHub release has matching git tag
2. Try: `GOPROXY=direct go install github.com/Adrian95/graphite-tui@latest`
3. Verify with: `go version -m $(which graphite-tui)`

**"go install gets wrong version":**
```bash
# Clear Go module cache and reinstall
go clean -modcache
GOPROXY=direct go install github.com/Adrian95/graphite-tui@v1.9.4
```

**Diagnostic function available:**
```go
config.DiagnoseVersionIssue(expected, actual)
// Returns possible causes: proxy cache, missing tag, etc.
```

## Common Patterns

### Adding a New Modal Dialog

```go
// 1. Add state in internal/state/states.go
const MyDialog StateID = "my_dialog"

// 2. Add view data struct in internal/ui/views/
type MyDialogViewData struct { ... }

// 3. Add render function
func RenderMyDialog(data MyDialogViewData) string { ... }

// 4. Handle keys in main.go
func (m model) handleMyDialogKeys(key string) (tea.Model, tea.Cmd) { ... }

// 5. Add to Update() switch
case state.MyDialog:
    return m.handleMyDialogKeys(key)

// 6. Add to View() switch
case state.MyDialog:
    return views.RenderCentered(views.RenderMyDialog(m.myDialogData))
```

### Executing a Graphite Command

```go
// 1. Add wrapper in internal/git/commands.go
func ExecuteMyCommand(skipHooks bool) tea.Cmd {
    cmd := "gt my-command --no-interactive"
    return ExecuteAsync(cmd, 30*time.Second)
}

// 2. Call from key handler
case "k":
    m.stateID = state.Running
    m.outputData = views.OutputViewData{IsRunning: true, Command: "my-command"}
    return m, git.ExecuteMyCommand(m.skipHooks)
```

## External Resources

- [Bubble Tea Documentation](https://pkg.go.dev/github.com/charmbracelet/bubbletea)
- [Bubble Tea Examples](https://github.com/charmbracelet/bubbletea/tree/master/examples)
- [Lipgloss Documentation](https://pkg.go.dev/github.com/charmbracelet/lipgloss)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Graphite CLI Docs](https://graphite.com/docs/cli-quick-start)
- [Graphite Stacking Guide](https://graphite.com/features/cli)

## Notes for AI Assistants

1. **Read before modifying:** Always read files before suggesting changes
2. **Follow MVU pattern:** State changes in Update(), rendering in View()
3. **Use existing styles:** Import from `internal/ui/ui.go`
4. **Test manually:** Run `go build && ./graphite-tui` in a git repo
5. **Keep it simple:** This is a power-user tool - minimal UI, maximum efficiency
6. **Conventional commits:** Use `feat:`, `fix:`, `docs:`, etc.
7. **No tests yet:** Be careful with refactoring without test coverage
