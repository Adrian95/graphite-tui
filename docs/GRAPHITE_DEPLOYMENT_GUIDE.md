# Graphite Deployment Guide

> **Fast, Simple SDLC for Teams of 5** — A comprehensive guide to deploying preview builds and production with stacked PRs.

This guide distills 10 best-in-class DX practices for teams using Graphite CLI to ship faster with confidence.

---

## Table of Contents

1. [Quick Start: Team Setup](#quick-start-team-setup)
2. [Essential Commands Reference](#essential-commands-reference)
3. [The 5-Minute Daily Workflow](#the-5-minute-daily-workflow)
4. [Preview Deployments](#preview-deployments)
5. [Production Deployment Flow](#production-deployment-flow)
6. [CI/CD Optimization](#cicd-optimization)
7. [Merge Queue Configuration](#merge-queue-configuration)
8. [Team Collaboration Patterns](#team-collaboration-patterns)
9. [Troubleshooting](#troubleshooting)
10. [DX Best Practices](#dx-best-practices)

---

## Quick Start: Team Setup

### Prerequisites

```bash
# Install Graphite CLI
npm install -g @withgraphite/graphite-cli

# Or via Homebrew (macOS)
brew install withgraphite/tap/graphite
```

### Team-Wide Git Configuration

Every team member should run these once:

```bash
# Enable conflict resolution memory (game-changer for stacks)
git config --global rerere.enabled true
git config --global rerere.autoupdate true

# Optimize for trunk-based development
git config --global pull.rebase true
git config --global rebase.autoStash true
```

### Repository Initialization

```bash
# Initialize Graphite in your repo
gt repo init --trunk main

# Authenticate with GitHub
gt auth
```

### Team Configuration File

Create `.graphite/team-config.yml` in your repo root:

```yaml
# .graphite/team-config.yml
branch_naming:
  prefix: "team"           # Branches: team/feature-name
  max_length: 50

stack_limits:
  max_depth: 5             # Keep stacks manageable for a team of 5
  warn_at: 3               # Gentle reminder to ship

merge_settings:
  auto_restack: true       # Auto-rebase after merges
  delete_on_merge: true    # Clean up merged branches

ci:
  skip_middle_prs: true    # Only run CI on top/bottom of stack
```

---

## Essential Commands Reference

### Core Workflow Commands

| Command | Description | When to Use |
|---------|-------------|-------------|
| `gt create -m "message"` | Create branch + commit | Starting new work |
| `gt create -a -m "msg"` | Create with all changes | Quick commits |
| `gt modify -m "message"` | Amend current commit | Iterating on feedback |
| `gt modify -a -m "msg"` | Amend with all changes | Quick amendments |
| `gt submit` | Push & create/update PR | Ready for review |
| `gt submit --stack` | Submit entire stack | Ship multiple PRs |
| `gt sync` | Sync with remote | Start of day / after merges |

### Stack Management

| Command | Description |
|---------|-------------|
| `gt log short` | Visualize your stack |
| `gt stack restack` | Rebase entire stack on trunk |
| `gt checkout <branch>` | Switch branches |
| `gt checkout --trunk` | Return to main |
| `gt branch delete <name>` | Clean up branch |
| `gt undo` | Reverse last operation |

### Advanced Operations

| Command | Description |
|---------|-------------|
| `gt submit --ai` | Auto-generate PR title/description |
| `gt submit --draft` | Create as draft PR |
| `gt submit --update-only` | Only update existing PRs |
| `gt get <branch>` | Checkout teammate's stack |
| `gt repo sync --restack` | Full sync + restack |

### Useful Aliases

Add to your shell profile:

```bash
alias gts="gt submit --stack"        # Submit full stack
alias gtc="gt create -a -m"          # Quick create
alias gtm="gt modify -a -m"          # Quick modify
alias gtl="gt log short"             # View stack
alias gty="gt sync"                  # Sync
```

---

## The 5-Minute Daily Workflow

### Morning Sync (30 seconds)

```bash
gt sync
```

This single command:
- Fetches latest from remote
- Detects merged/closed PRs
- Prompts to delete stale branches
- Restacks your work on latest trunk

### Feature Development Loop

```bash
# 1. Start feature (creates branch + commit)
gt create -a -m "feat: add user authentication"

# 2. Continue building (stacked on previous)
gt create -a -m "feat: add login form UI"

# 3. Add another layer
gt create -a -m "feat: add session management"

# 4. Submit entire stack for review
gt submit --stack
```

### Iteration After Review

```bash
# Make changes based on feedback
gt modify -a -m "feat: add user authentication

- Added password validation
- Fixed edge case for empty email"

# Push updates
gt submit
```

### Merging & Cleanup

```bash
# After approval, merge from Graphite dashboard or:
gt merge

# Sync to clean up
gt sync
```

---

## Preview Deployments

### Architecture Overview

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  gt submit  │────▶│  GitHub PR   │────▶│ Preview Deploy  │
│  --stack    │     │  Created     │     │ (per PR)        │
└─────────────┘     └──────────────┘     └─────────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  CI Checks   │
                    │  Run on PR   │
                    └──────────────┘
```

### GitHub Actions for Preview Deployments

Create `.github/workflows/preview.yml`:

```yaml
name: Preview Deployment

on:
  pull_request:
    types: [opened, synchronize, reopened]

jobs:
  deploy-preview:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci

      - name: Build
        run: npm run build
        env:
          PREVIEW_URL: "https://preview-${{ github.event.pull_request.number }}.example.com"

      - name: Deploy to Preview
        id: deploy
        run: |
          # Your deployment command (Vercel, Netlify, etc.)
          echo "preview_url=https://preview-${{ github.event.pull_request.number }}.example.com" >> $GITHUB_OUTPUT

      - name: Comment Preview URL
        uses: actions/github-script@v7
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: '🚀 Preview deployed: ${{ steps.deploy.outputs.preview_url }}'
            })
```

### Vercel Integration (Zero-Config)

```yaml
# .github/workflows/vercel-preview.yml
name: Vercel Preview

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Deploy to Vercel
        uses: amondnet/vercel-action@v25
        with:
          vercel-token: ${{ secrets.VERCEL_TOKEN }}
          vercel-org-id: ${{ secrets.VERCEL_ORG_ID }}
          vercel-project-id: ${{ secrets.VERCEL_PROJECT_ID }}
          # Creates unique preview URL per PR
```

### Preview Per Stack Level

For stacked PRs, each PR in the stack gets its own preview:

```yaml
# .github/workflows/stack-preview.yml
name: Stack Preview

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  check-stack-position:
    runs-on: ubuntu-latest
    outputs:
      is_top: ${{ steps.check.outputs.is_top }}
    steps:
      - name: Check Stack Position
        id: check
        run: |
          # Query Graphite API to check if this is top of stack
          RESPONSE=$(curl -s -H "Authorization: Bearer ${{ secrets.GRAPHITE_TOKEN }}" \
            "https://api.graphite.dev/v1/prs/${{ github.event.pull_request.number }}/stack")
          IS_TOP=$(echo $RESPONSE | jq -r '.is_top_of_stack')
          echo "is_top=$IS_TOP" >> $GITHUB_OUTPUT

  deploy-preview:
    needs: check-stack-position
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Deploy Preview
        run: |
          # Deploy with stack-aware naming
          PREVIEW_NAME="pr-${{ github.event.pull_request.number }}"
          echo "Deploying $PREVIEW_NAME"
          # Your deploy command
```

---

## Production Deployment Flow

### Recommended Flow for Team of 5

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Develop   │────▶│   Review    │────▶│ Merge Queue │────▶│ Production  │
│  (Stack)    │     │  (Preview)  │     │  (Validate) │     │  (Deploy)   │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
      │                   │                   │                   │
      ▼                   ▼                   ▼                   ▼
  gt submit           Preview URL        CI + Rebase         Auto-deploy
   --stack            per PR             on trunk            to prod
```

### Production Deployment Workflow

```yaml
# .github/workflows/production.yml
name: Production Deployment

on:
  push:
    branches: [main]

concurrency:
  group: production
  cancel-in-progress: false  # Never cancel prod deploys

jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: production
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'

      - name: Install & Build
        run: |
          npm ci
          npm run build

      - name: Run Tests
        run: npm test

      - name: Deploy to Production
        run: |
          # Your production deploy command
          npm run deploy:prod
        env:
          DEPLOY_TOKEN: ${{ secrets.DEPLOY_TOKEN }}

      - name: Notify Team
        if: success()
        run: |
          curl -X POST ${{ secrets.SLACK_WEBHOOK }} \
            -H 'Content-Type: application/json' \
            -d '{"text":"✅ Deployed to production: ${{ github.sha }}"}'

      - name: Rollback on Failure
        if: failure()
        run: |
          echo "Deployment failed - triggering rollback"
          # Your rollback command
```

### Hotfix Flow

When you need to ship a fix immediately:

```bash
# 1. Switch to trunk
gt checkout --trunk

# 2. Create hotfix branch
gt create -a -m "fix: critical auth bug"

# 3. Submit immediately (bypasses stack)
gt submit

# 4. Add to merge queue with priority
# (Via Graphite dashboard - mark as urgent)

# 5. After merge, sync other stacks
gt sync
```

---

## CI/CD Optimization

### The CI Cost Problem with Stacks

Stacking creates more PRs, which can mean more CI runs. Here's how to optimize:

### Strategy 1: Run CI Only on Stack Edges

```yaml
# .github/workflows/optimized-ci.yml
name: Optimized CI

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  check-stack:
    runs-on: ubuntu-latest
    outputs:
      should_run: ${{ steps.check.outputs.should_run }}
    steps:
      - name: Check Graphite Stack Position
        id: check
        run: |
          # Graphite API to check stack position
          # Only run full CI on top and bottom of stack
          RESPONSE=$(curl -s \
            -H "Authorization: Bearer ${{ secrets.GRAPHITE_TOKEN }}" \
            "https://api.graphite.dev/v1/ci-check?pr=${{ github.event.pull_request.number }}")

          SHOULD_RUN=$(echo $RESPONSE | jq -r '.run_ci')
          echo "should_run=$SHOULD_RUN" >> $GITHUB_OUTPUT

  full-ci:
    needs: check-stack
    if: needs.check-stack.outputs.should_run == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: npm ci
      - run: npm test
      - run: npm run build

  minimal-ci:
    needs: check-stack
    if: needs.check-stack.outputs.should_run == 'false'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: npm ci
      - run: npm run lint  # Just lint, skip heavy tests
```

### Strategy 2: Use Turborepo/Nx Caching

```yaml
# .github/workflows/cached-ci.yml
name: Cached CI

on:
  pull_request:

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for cache detection

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'

      - name: Turborepo Cache
        uses: actions/cache@v4
        with:
          path: .turbo
          key: turbo-${{ github.sha }}
          restore-keys: |
            turbo-

      - run: npm ci

      - name: Build with Turbo
        run: npx turbo build test --filter='...[origin/main]'
        # Only builds/tests what changed since main
```

### Strategy 3: Parallel Test Sharding

```yaml
# .github/workflows/parallel-tests.yml
name: Parallel Tests

on:
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        shard: [1, 2, 3, 4]  # 4 parallel runners
    steps:
      - uses: actions/checkout@v4
      - run: npm ci
      - run: npm test -- --shard=${{ matrix.shard }}/4
```

---

## Merge Queue Configuration

### Setting Up Graphite Merge Queue

1. **Install Graphite GitHub App** on your repository
2. **Disable GitHub's merge queue** (they're incompatible)
3. **Configure branch protection** to require Graphite

### Branch Protection Settings

```
Repository Settings → Branches → Branch protection rules → main:

☑ Require a pull request before merging
☑ Require approvals: 1
☑ Require status checks to pass
  - ci/build
  - ci/test
☑ Require branches to be up to date
☐ Require merge queue (use Graphite instead)
☑ Require linear history
```

### Graphite Merge Queue Settings

Via Graphite Dashboard → Repository Settings:

```yaml
merge_queue:
  # Merge strategy
  strategy: squash          # or: merge, rebase

  # Batching (reduces CI runs)
  batch_size: 3             # Merge up to 3 PRs together
  batch_timeout: 300        # Wait 5 min to batch

  # Stack handling
  stack_aware: true         # Process stacks together

  # Required checks
  required_checks:
    - ci/build
    - ci/test
    - preview/deploy

  # Auto-merge when ready
  auto_merge: true

  # Parallel validation
  parallel_mode: optimistic # Assume success, validate in parallel
```

### Merge Queue GitHub Action Hook

```yaml
# .github/workflows/merge-queue.yml
name: Merge Queue Validation

on:
  merge_group:
    types: [checks_requested]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Full Test Suite
        run: |
          npm ci
          npm run test:all
          npm run build

      - name: Integration Tests
        run: npm run test:integration
```

---

## Team Collaboration Patterns

### Pattern 1: Feature Leads

For a team of 5, assign feature leads who own stacks:

```
Alice (Lead: Auth)     Bob (Lead: API)      Carol (Lead: UI)
      │                     │                    │
      ▼                     ▼                    ▼
  auth/base             api/base             ui/base
      │                     │                    │
  auth/login            api/users            ui/forms
      │                     │                    │
  auth/sessions         api/posts            ui/dashboard
```

### Pattern 2: Review Handoffs

```bash
# Alice creates foundation
gt create -a -m "feat: auth base infrastructure"
gt submit

# Bob builds on Alice's work
gt get alice/auth-base           # Checkout Alice's stack
gt create -a -m "feat: add API auth endpoints"
gt submit
```

### Pattern 3: Stack Depth Limits

Keep stacks at 3-5 PRs max for a team of 5:

```
Good:                          Avoid:
  main                           main
    │                              │
  feat-1 ← Review                feat-1
    │                              │
  feat-2 ← Review                feat-2
    │                              │
  feat-3 ← Review                feat-3
                                   │
                                 feat-4
                                   │
                                 feat-5
                                   │
                                 feat-6 ← Too deep!
```

### Pattern 4: Daily Standup Integration

```bash
# Show your current work
gt log short

# Example output:
◉ feat/session-management (current)
│ PR #45: feat: add session management
│
◉ feat/login-form
│ PR #44: feat: add login form UI
│
◉ feat/auth-base
│ PR #43: feat: add user authentication
│
◯ main
```

---

## Troubleshooting

### Common Issues

#### "Stack out of sync with remote"

```bash
gt repo sync --restack
```

#### "Merge conflicts in stack"

```bash
# Fix conflicts one at a time
gt stack fix --one-at-a-time

# Or let git remember resolutions
git config rerere.enabled true
gt stack restack
```

#### "PR shows wrong diff"

```bash
# Force update the PR
gt submit --force
```

#### "Teammate's changes not showing"

```bash
# Fetch their stack
gt get teammate/branch-name

# Or full sync
gt sync
```

#### "CI running on every PR in stack"

Configure CI optimization (see [CI/CD Optimization](#cicd-optimization))

### Emergency Procedures

#### Rollback a Merged PR

```bash
# Create revert PR
gt create --revert PR_NUMBER
gt submit
```

#### Abandon a Stack

```bash
# Switch to trunk
gt checkout --trunk

# Delete the stack branches
gt branch delete feature-branch --force
```

#### Hotfix While Mid-Stack

```bash
# Save current position
CURRENT=$(git branch --show-current)

# Create hotfix from trunk
gt checkout --trunk
gt create -a -m "fix: critical bug"
gt submit

# Return to your stack
gt checkout $CURRENT
gt sync
```

---

## DX Best Practices

### 1. Ship Small, Ship Often

```bash
# Bad: One massive PR
gt create -a -m "feat: entire authentication system"

# Good: Stacked small PRs
gt create -a -m "feat: add user model"
gt create -a -m "feat: add auth middleware"
gt create -a -m "feat: add login endpoint"
gt create -a -m "feat: add session management"
gt submit --stack
```

### 2. Descriptive Commit Messages

```bash
# Use conventional commits
gt create -a -m "feat(auth): add JWT token validation

- Validate token expiration
- Check token signature
- Handle refresh tokens"
```

### 3. Review from Bottom Up

When reviewing stacked PRs:
1. Start at the bottom of the stack
2. Review foundational changes first
3. Work your way up
4. Approve the entire stack when ready

### 4. Sync Early, Sync Often

```bash
# Start every session with
gt sync

# After any teammate merges
gt sync
```

### 5. Use AI-Generated PR Descriptions

```bash
# Let AI draft your PR
gt submit --ai

# Review and edit before publishing
```

### 6. Keyboard-First Workflow

If using graphite-tui:
- `s` - Start commit wizard
- `p` - Submit PR
- `f` - Iterate (amend + push)
- `y` - Sync
- `d` - Merge + cleanup
- `g` - View stack

### 7. Automate the Boring Stuff

```bash
# Shell function for full workflow
ship() {
  gt sync && \
  gt create -a -m "$1" && \
  gt submit
}

# Usage: ship "feat: add new feature"
```

### 8. Keep Main Green

- Never merge failing PRs
- Use merge queue to validate
- Roll back immediately if prod breaks

### 9. Document Stack Intent

In PR descriptions, explain how this PR fits in the stack:

```markdown
## Stack Context
This PR is part 2 of 3 in the auth stack:
1. ✅ #43 - User model
2. 👉 #44 - Auth middleware (this PR)
3. ⏳ #45 - Login endpoint
```

### 10. Celebrate Velocity

Track your team's shipping metrics:
- PRs merged per week
- Time from PR open to merge
- Stack completion rate

---

## Quick Reference Card

```
┌─────────────────────────────────────────────────────────────┐
│                    GRAPHITE QUICK REFERENCE                 │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  DAILY WORKFLOW                                             │
│  ─────────────                                              │
│  gt sync              Sync with remote (do this first!)    │
│  gt create -a -m ""   Create branch + commit               │
│  gt modify -a -m ""   Amend current commit                 │
│  gt submit --stack    Submit entire stack for review       │
│                                                             │
│  NAVIGATION                                                 │
│  ──────────                                                 │
│  gt log short         View your stack                      │
│  gt checkout <branch> Switch branches                      │
│  gt checkout --trunk  Return to main                       │
│                                                             │
│  COLLABORATION                                              │
│  ─────────────                                              │
│  gt get <branch>      Checkout teammate's stack            │
│  gt stack restack     Rebase stack on trunk                │
│  gt undo              Reverse last operation               │
│                                                             │
│  SHIPPING                                                   │
│  ────────                                                   │
│  gt submit            Push & create/update PR              │
│  gt merge             Merge via Graphite                   │
│  gt sync              Clean up after merge                 │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Resources

- [Graphite Documentation](https://graphite.dev/docs)
- [Graphite CLI Quick Start](https://graphite.com/docs/cli-quick-start)
- [Stacked PRs Guide](https://graphite.com/blog/stacked-prs)
- [CI Optimizations](https://graphite.dev/docs/stacking-and-ci)
- [Merge Queue Setup](https://graphite.dev/docs/set-up-merge-queue)

---

*Last updated: January 2026*
