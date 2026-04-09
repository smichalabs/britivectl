package britive

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// CheckedOutProfile is an active checkout returned by GET /api/access/app-access-status.
type CheckedOutProfile struct {
	TransactionID  string  `json:"transactionId"`
	PapID          string  `json:"papId"` // = profileId
	EnvironmentID  string  `json:"environmentId"`
	AppContainerID string  `json:"appContainerId"`
	AccessType     string  `json:"accessType"`
	Status         string  `json:"status"`
	CheckedOut     string  `json:"checkedOut"`
	Expiration     string  `json:"expiration"`
	CheckedIn      *string `json:"checkedIn"`
}

// Transaction is returned by the checkout POST.
type Transaction struct {
	TransactionID  string `json:"transactionId"`
	ProfileID      string `json:"profileId"`
	EnvironmentID  string `json:"environmentId"`
	Status         string `json:"status"`
	AccessType     string `json:"accessType"`
	ExpirationTime string `json:"expirationTime"`
}

// Credentials holds the temporary cloud credentials from a checkout.
type Credentials struct {
	// AWS
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken"`
	Region          string `json:"region"`
	Expiration      string `json:"expiration"`
}

// Checkout checks out a profile and returns credentials.
// Flow: POST /api/access/{profileId}/environments/{environmentId}?accessType=PROGRAMMATIC
//
//	→ poll until status=checkedOut
//	→ GET /api/access/{transactionId}/tokens
func (c *Client) Checkout(profileID, environmentID string) (*CheckedOutProfile, *Credentials, error) {
	if profileID == "" || environmentID == "" {
		return nil, nil, fmt.Errorf("profileId and environmentId must not be empty")
	}

	// Initiate checkout
	var txn Transaction
	path := fmt.Sprintf("/api/access/%s/environments/%s?accessType=PROGRAMMATIC", profileID, environmentID)
	if err := c.post(path, map[string]string{}, &txn); err != nil {
		return nil, nil, fmt.Errorf("checkout initiation failed: %w", err)
	}

	// Poll until checkedOut (async checkout may take a few seconds)
	transactionID := txn.TransactionID
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		active, err := c.MySessions()
		if err != nil {
			return nil, nil, err
		}
		for _, p := range active {
			if p.TransactionID == transactionID && p.Status == "checkedOut" {
				creds, err := c.GetCredentials(transactionID)
				if err != nil {
					return nil, nil, err
				}
				return &p, creds, nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return nil, nil, fmt.Errorf("checkout timed out after 2 minutes")
}

// GetCredentials retrieves credentials for an active checkout.
// GET /api/access/{transactionId}/tokens
func (c *Client) GetCredentials(transactionID string) (*Credentials, error) {
	var creds Credentials
	if err := c.get(fmt.Sprintf("/api/access/%s/tokens", transactionID), &creds); err != nil {
		return nil, fmt.Errorf("fetching credentials: %w", err)
	}
	return &creds, nil
}

// Checkin returns a checked-out profile early.
// PUT /api/access/{transactionId}?type=API
func (c *Client) Checkin(transactionID string) error {
	if transactionID == "" {
		return fmt.Errorf("transactionId must not be empty")
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut,
		fmt.Sprintf("%s/api/access/%s?type=API", c.baseURL, transactionID), nil)
	if err != nil {
		return fmt.Errorf("creating checkin request: %w", err)
	}
	c.setHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("checkin failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("checkin returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// MySessions returns all currently active profile checkouts.
// GET /api/access/app-access-status
func (c *Client) MySessions() ([]CheckedOutProfile, error) {
	var profiles []CheckedOutProfile
	if err := c.get("/api/access/app-access-status", &profiles); err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	return profiles, nil
}
