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

		if len(branches) == 0 {
			branches = []string{"production"}
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

		matchedPreview := false
		matchedProduction := false

		updateLatest := func(target string, dep Deployment) {
			branch := dep.GitSource.Ref
			if branch == "" {
				branch = dep.Meta["githubCommitRef"]
			}
			if branch == "" {
				branch = dep.Meta["gitBranch"]
			}
			if branch == "" {
				return
			}
			if _, ok := statusMap[branch]; !ok {
				return
			}
			created := dep.CreatedAt
			if created == 0 {
				created = dep.Created
			}
			if target == "production" {
				matchedProduction = true
				current := statusMap[branch].Production
				currentCreated := int64(0)
				if current != nil {
					currentCreated = current.CreatedAt
					if currentCreated == 0 {
						currentCreated = current.Created
					}
				}
				if current == nil || created > currentCreated {
					copy := dep
					statusMap[branch].Production = &copy
					statusMap[branch].ProductionURL = "https://" + dep.URL
				}
				return
			}
			matchedPreview = true
			current := statusMap[branch].Preview
			currentCreated := int64(0)
			if current != nil {
				currentCreated = current.CreatedAt
				if currentCreated == 0 {
					currentCreated = current.Created
				}
			}
			if current == nil || created > currentCreated {
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

		// If we didn't match any branches, fallback to latest deployments
		if len(statuses) == 0 || !matchedPreview || !matchedProduction {
			fallback := DeploymentStatus{Branch: "production"}
			var latestProd int64
			for _, dep := range prodResp.Deployments {
				created := dep.CreatedAt
				if created == 0 {
					created = dep.Created
				}
				if fallback.Production == nil || created > latestProd {
					latestProd = created
					copy := dep
					fallback.Production = &copy
					fallback.ProductionURL = "https://" + dep.URL
				}
			}
			var latestPreview int64
			for _, dep := range previewResp.Deployments {
				if strings.ToLower(dep.Target) == "production" {
					continue
				}
				created := dep.CreatedAt
				if created == 0 {
					created = dep.Created
				}
				if fallback.Preview == nil || created > latestPreview {
					latestPreview = created
					copy := dep
					fallback.Preview = &copy
					fallback.PreviewURL = "https://" + dep.URL
				}
			}
			if fallback.Production != nil || fallback.Preview != nil {
				statuses = append(statuses, fallback)
			}
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
