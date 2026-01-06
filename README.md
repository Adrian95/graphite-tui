# Graphite TUI Speedrun

A simple, fast TUI for the [Graphite](https://graphite.dev) workflow, written in Go.
Designed to help you speedrun your PRs with the "Solo Speedrun" workflow.

## Features

- **Start**: Create branch & commit (`gt c -am`)
- **Preview**: Push & Open PR (`gt s`)
- **Fix**: Amend changes (`gt m -a`)
- **Done**: Merge & Cleanup (`gt merge && gt sync`)
- **Rescue Kit**: Fix ghost branches, fold stacks, and undo commands.

## Installation

### From Source

Requires Go 1.20+.

```bash
git clone https://github.com/adrian/graphite-tui.git
cd graphite-tui
go install
```

Ensure your `$(go env GOPATH)/bin` is in your `$PATH`.

## Usage

Run the tool from anywhere in your terminal:

```bash
graphite-tui
```

(Or rename the binary to `gtt` for speed!)

### Keybindings

- **Up/Down (j/k)**: Navigate menu
- **Enter**: Select action
- **Esc**: Go back / Cancel
- **q**: Quit

## Prerequisites

- [Graphite CLI](https://graphite.dev/docs/cli/installation) (`gt`) must be installed and authenticated.
