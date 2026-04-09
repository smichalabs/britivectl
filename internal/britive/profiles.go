package britive

// Profile represents a Britive access profile.
type Profile struct {
	ID          string `json:"papId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	AppName     string `json:"appName"`
	EnvName     string `json:"envName"`
}

// ListProfiles returns all profiles available to the current user.
// GET /api/v1/my-access/profiles
func (c *Client) ListProfiles() ([]Profile, error) {
	var result struct {
		Data []Profile `json:"data"`
	}
	if err := c.get("/api/v1/my-access/profiles", &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}
