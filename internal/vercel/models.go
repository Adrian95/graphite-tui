package vercel

// Deployment represents a Vercel deployment
// API reference: https://vercel.com/docs/rest-api/endpoints/deployments
type Deployment struct {
	UID       string            `json:"uid"`
	Name      string            `json:"name"`
	URL       string            `json:"url"`
	State     string            `json:"state"` // BUILDING, READY, ERROR
	Created   int64             `json:"created"`
	CreatedAt int64             `json:"createdAt"`
	Target    string            `json:"target"` // preview or production
	Meta      map[string]string `json:"meta"`
	GitSource struct {
		Ref string `json:"ref"` // branch name
	} `json:"gitSource"`
}

// DeploymentsResponse represents response for deployments list
type DeploymentsResponse struct {
	Deployments []Deployment `json:"deployments"`
}

// DeploymentStatus aggregates preview + production info
type DeploymentStatus struct {
	Branch        string
	Preview       *Deployment
	Production    *Deployment
	PreviewURL    string
	ProductionURL string
}
