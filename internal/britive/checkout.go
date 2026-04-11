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
//	→ poll until status=checkedOut (respects ctx cancellation)
//	→ GET /api/access/{transactionId}/tokens
//
// Returns ErrCheckoutTimeout (wrapped) if the context deadline is reached
// before the checkout completes.
func (c *Client) Checkout(ctx context.Context, profileID, environmentID string) (*CheckedOutProfile, *Credentials, error) {
	if profileID == "" || environmentID == "" {
		return nil, nil, fmt.Errorf("profileId and environmentId must not be empty")
	}

	// Initiate checkout
	var txn Transaction
	path := fmt.Sprintf("/api/access/%s/environments/%s?accessType=PROGRAMMATIC", profileID, environmentID)
	if err := c.post(ctx, path, map[string]string{}, &txn); err != nil {
		return nil, nil, fmt.Errorf("checkout initiation failed: %w", err)
	}

	// Default to a 2 minute deadline if the caller did not set one.
	transactionID := txn.TransactionID
	pollCtx, cancel := contextWithDefaultDeadline(ctx, 2*time.Minute)
	defer cancel()

	for {
		active, err := c.MySessions(pollCtx)
		if err != nil {
			return nil, nil, err
		}
		for _, p := range active {
			if p.TransactionID == transactionID && p.Status == "checkedOut" {
				creds, err := c.GetCredentials(pollCtx, transactionID)
				if err != nil {
					return nil, nil, err
				}
				return &p, creds, nil
			}
		}

		select {
		case <-pollCtx.Done():
			return nil, nil, fmt.Errorf("%w: %w", ErrCheckoutTimeout, pollCtx.Err())
		case <-time.After(2 * time.Second):
			// continue polling
		}
	}
}

// GetCredentials retrieves credentials for an active checkout.
// GET /api/access/{transactionId}/tokens
func (c *Client) GetCredentials(ctx context.Context, transactionID string) (*Credentials, error) {
	var creds Credentials
	if err := c.get(ctx, fmt.Sprintf("/api/access/%s/tokens", transactionID), &creds); err != nil {
		return nil, fmt.Errorf("fetching credentials: %w", err)
	}
	return &creds, nil
}

// Checkin returns a checked-out profile early.
// PUT /api/access/{transactionId}?type=API
func (c *Client) Checkin(ctx context.Context, transactionID string) error {
	if transactionID == "" {
		return fmt.Errorf("transactionId must not be empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
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
func (c *Client) MySessions(ctx context.Context) ([]CheckedOutProfile, error) {
	var profiles []CheckedOutProfile
	if err := c.get(ctx, "/api/access/app-access-status", &profiles); err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	return profiles, nil
}

// contextWithDefaultDeadline returns a derived context that enforces the given
// timeout only if the parent context has no existing deadline. This lets
// callers override the default (e.g. shorter timeouts in tests) while keeping
// sensible behavior for unbounded parents.
func contextWithDefaultDeadline(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := parent.Deadline(); ok {
		return parent, func() {}
	}
	return context.WithTimeout(parent, timeout)
}
