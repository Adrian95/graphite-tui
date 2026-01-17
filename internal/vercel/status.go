package vercel

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Summary represents aggregated deployment status counts
type Summary struct {
	PreviewReady    int
	PreviewBuilding int
	PreviewError    int
	ProdReady       int
	ProdBuilding    int
	ProdError       int
}

// StatusMsg carries deployment status data to the UI
type StatusMsg struct {
	Enabled  bool
	Statuses []DeploymentStatus
	Summary  Summary
	Err      error
}

// ActionMsg is used for simple user feedback actions (open/copy)
type ActionMsg struct {
	Message string
	Err     error
}

// FetchStatus fetches deployments and builds status for the given branches
func FetchStatus(client *Client, branches []string) tea.Cmd {
	return func() tea.Msg {
		if client == nil || !client.config.Enabled() {
			return StatusMsg{Enabled: false}
		}

		previewResp, err := client.GetDeployments(50, "")
		if err != nil {
			return StatusMsg{Enabled: true, Err: err}
		}

		prodResp, err := client.GetDeployments(30, "production")
		if err != nil {
			return StatusMsg{Enabled: true, Err: err}
		}

		statusMap := map[string]*DeploymentStatus{}
		for _, branch := range branches {
			if branch == "" {
				continue
			}
			statusMap[branch] = &DeploymentStatus{Branch: branch}
		}

		updateLatest := func(target string, dep Deployment) {
			branch := dep.GitSource.Ref
			if branch == "" {
				return
			}
			if _, ok := statusMap[branch]; !ok {
				return
			}
			if target == "production" {
				current := statusMap[branch].Production
				if current == nil || dep.CreatedAt > current.CreatedAt {
					copy := dep
					statusMap[branch].Production = &copy
					statusMap[branch].ProductionURL = "https://" + dep.URL
				}
				return
			}
			current := statusMap[branch].Preview
			if current == nil || dep.CreatedAt > current.CreatedAt {
				copy := dep
				statusMap[branch].Preview = &copy
				statusMap[branch].PreviewURL = "https://" + dep.URL
			}
		}

		for _, dep := range previewResp.Deployments {
			if strings.ToLower(dep.Target) == "production" {
				continue
			}
			updateLatest("preview", dep)
		}

		for _, dep := range prodResp.Deployments {
			updateLatest("production", dep)
		}

		statuses := make([]DeploymentStatus, 0, len(statusMap))
		for _, status := range statusMap {
			statuses = append(statuses, *status)
		}

		sort.Slice(statuses, func(i, j int) bool {
			return statuses[i].Branch < statuses[j].Branch
		})

		summary := summarize(statuses)

		return StatusMsg{
			Enabled:  true,
			Statuses: statuses,
			Summary:  summary,
		}
	}
}

func summarize(statuses []DeploymentStatus) Summary {
	summary := Summary{}
	for _, status := range statuses {
		if status.Preview != nil {
			switch classifyState(status.Preview.State) {
			case "ready":
				summary.PreviewReady++
			case "error":
				summary.PreviewError++
			default:
				summary.PreviewBuilding++
			}
		}
		if status.Production != nil {
			switch classifyState(status.Production.State) {
			case "ready":
				summary.ProdReady++
			case "error":
				summary.ProdError++
			default:
				summary.ProdBuilding++
			}
		}
	}
	return summary
}

func classifyState(state string) string {
	switch strings.ToLower(state) {
	case "ready":
		return "ready"
	case "error":
		return "error"
	default:
		return "building"
	}
}
