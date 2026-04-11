package britive

import "context"

// AppAccess is the top-level entry returned by GET /api/access.
// Each entry is an application with embedded profiles and environments.
type AppAccess struct {
	AppContainerID string `json:"appContainerId"`
	AppName        string `json:"appName"`
	CatalogAppName string `json:"catalogAppName"`
	Profiles       []PAP  `json:"profiles"`
}

// PAP is a Britive Profile (PAP = Profile Access Profile).
type PAP struct {
	ProfileID    string `json:"profileId"`
	ProfileName  string `json:"profileName"`
	PapID        int    `json:"papId"`
	Environments []Env  `json:"environments"`
}

// Env is a Britive environment (cloud account/project/subscription).
type Env struct {
	EnvironmentID   string `json:"environmentId"`
	EnvironmentName string `json:"environmentName"`
}

// AccessEntry is a flattened profile+environment combination ready for bctl use.
type AccessEntry struct {
	AppName         string
	AppContainerID  string
	ProfileName     string
	ProfileID       string
	PapID           int
	EnvironmentName string
	EnvironmentID   string
	Cloud           string // derived from catalogAppName
}

// ListAccess returns all profiles available to the current user, flattened.
// GET /api/access
func (c *Client) ListAccess(ctx context.Context) ([]AccessEntry, error) {
	var apps []AppAccess
	if err := c.get(ctx, "/api/access", &apps); err != nil {
		return nil, err
	}

	var entries []AccessEntry
	for _, app := range apps {
		cloud := catalogToCloud(app.CatalogAppName)
		for _, pap := range app.Profiles {
			for _, env := range pap.Environments {
				entries = append(entries, AccessEntry{
					AppName:         app.AppName,
					AppContainerID:  app.AppContainerID,
					ProfileName:     pap.ProfileName,
					ProfileID:       pap.ProfileID,
					PapID:           pap.PapID,
					EnvironmentName: env.EnvironmentName,
					EnvironmentID:   env.EnvironmentID,
					Cloud:           cloud,
				})
			}
		}
	}
	return entries, nil
}

// catalogToCloud converts a Britive catalog app name to a short cloud identifier.
func catalogToCloud(catalog string) string {
	switch catalog {
	case "AWS", "AWS_STANDALONE":
		return "aws"
	case "GCP":
		return "gcp"
	case "Azure":
		return "azure"
	default:
		return "other"
	}
}
