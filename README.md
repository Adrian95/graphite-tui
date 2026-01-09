# Graphite TUI Speedrun

<p align="center">
  <img src="https://graphite.dev/favicon.ico" width="50" height="50" alt="Graphite Logo">
</p>

A **Vercel-inspired**, beautiful TUI for the [Graphite](https://graphite.dev) workflow.
Designed for the "Solo Speedrun" workflow, built with Go, Bubble Tea, and Lipgloss.

## ✨ Features

- **Beautiful UI**: Minimalist, high-contrast aesthetics compatible with modern terminals (Ghostty, Alacritty, iTerm2).
- **Speedrun Workflow**:
  - **Start**: `gt c -am`
  - **Preview**: `gt s`
  - **Fix**: `gt m -a`
  - **Done**: `gt merge && gt sync`
- **Rescue Kit**: Built-in tools to fix ghost branches and undo mistakes.
- **Fast**: Compiles to a single binary.

## 🛠️ Installation

### Quick Install (Recommended)

```bash
go install github.com/Adrian95/graphite-tui@latest
```

### From Source

```bash
git clone https://github.com/Adrian95/graphite-tui.git
cd graphite-tui
go install
```

Make sure your `$(go env GOPATH)/bin` is in your `$PATH`.

## 🚀 Usage

Run it from anywhere:

```bash
graphite-tui
```

### Keybindings

| Key | Action |
| :--- | :--- |
| **j / k** | Navigate Menu |
| **Enter** | Select / Confirm |
| **Esc** | Back / Cancel |
| **q** | Quit |

## 🎨 Design

The interface is designed to be clean and focused:
- **Dark Mode First**: Optimized for dark terminal themes.
- **Visual Feedback**: Clear status indicators for success/failure.
- **Focus**: Distraction-free input for commit messages.

## Prerequisites

- [Graphite CLI](https://graphite.dev/docs/cli/installation) (`gt`)
