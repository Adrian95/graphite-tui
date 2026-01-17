package vercel

import (
	"os"
	"strings"
)

// Config holds Vercel auth details
// One repo → one project
// Token is loaded from .env.local or .env
// Required: VERCEL_TOKEN, VERCEL_PROJECT_ID
// Optional: VERCEL_TEAM_ID
type Config struct {
	Token     string
	ProjectID string
	TeamID    string
}

// Enabled returns true if Vercel config is available
func (c Config) Enabled() bool {
	return c.Token != "" && c.ProjectID != ""
}

// LoadEnvFiles reads .env.local, .env and populates environment variables if not already set
func LoadEnvFiles(paths ...string) {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			if strings.HasPrefix(line, "export ") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, "\"'")

			if key == "" {
				continue
			}

			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, value)
			}
		}
	}
}

// LoadConfig reads Vercel config from env vars
func LoadConfig() Config {
	return Config{
		Token:     os.Getenv("VERCEL_TOKEN"),
		ProjectID: os.Getenv("VERCEL_PROJECT_ID"),
		TeamID:    os.Getenv("VERCEL_TEAM_ID"),
	}
}
