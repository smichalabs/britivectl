package britive

import "fmt"

// Session represents an active Britive profile checkout session.
type Session struct {
	SessionID       string      `json:"sessionId"`
	PapID           string      `json:"papId"`
	ProfileName     string      `json:"profileName"`
	Status          string      `json:"status"`
	CreatedAt       string      `json:"createdAt"`
	ExpiresAt       string      `json:"expiresAt"`
	Credentials     Credentials `json:"credentials"`
}

// Credentials holds cloud-provider credentials from a checkout.
type Credentials struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken"`
	Region          string `json:"region"`
	Expiration      string `json:"expiration"`
}

// Checkout checks out a Britive profile, returning the session with credentials.
// POST /api/v1/profile-session/{papId}/checkout
func (c *Client) Checkout(papID string) (*Session, error) {
	if papID == "" {
		return nil, fmt.Errorf("papId must not be empty")
	}
	var session Session
	if err := c.post(fmt.Sprintf("/api/v1/profile-session/%s/checkout", papID), nil, &session); err != nil {
		return nil, fmt.Errorf("checkout failed: %w", err)
	}
	return &session, nil
}

// Checkin returns a checked-out profile early.
// POST /api/v1/profile-session/{papId}/checkin
func (c *Client) Checkin(papID string) error {
	if papID == "" {
		return fmt.Errorf("papId must not be empty")
	}
	if err := c.post(fmt.Sprintf("/api/v1/profile-session/%s/checkin", papID), nil, nil); err != nil {
		return fmt.Errorf("checkin failed: %w", err)
	}
	return nil
}

// MySessions returns all active checkout sessions for the current user.
// GET /api/v1/profile-session/my-sessions
func (c *Client) MySessions() ([]Session, error) {
	var sessions []Session
	if err := c.get("/api/v1/profile-session/my-sessions", &sessions); err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	return sessions, nil
}
