package britive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Access types accepted by the Britive checkout endpoint. PROGRAMMATIC
// returns API credentials; CONSOLE returns a federated sign-in URL for the
// cloud provider's web console.
const (
	AccessTypeProgrammatic = "PROGRAMMATIC"
	AccessTypeConsole      = "CONSOLE"
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

// HasAccessType reports whether the session is of the given access type.
// Sessions without an access type are treated as programmatic, matching the
// behavior of checkouts made before console access was supported.
func (p CheckedOutProfile) HasAccessType(accessType string) bool {
	if p.AccessType == "" {
		return accessType == AccessTypeProgrammatic
	}
	return p.AccessType == accessType
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

// maxConsecutivePollFailures caps how many back-to-back MySessions errors we
// will tolerate during a checkout. Transient blips (a dropped connection, a
// momentary 502) should not kill an otherwise-in-progress checkout; a genuine
// outage should still give up eventually instead of spinning until the
// overall deadline. Five retries at 2s each costs at most ~10s.
const maxConsecutivePollFailures = 5

// Checkout checks out a profile for programmatic access and returns credentials.
// Flow: POST /api/access/{profileId}/environments/{environmentId}?accessType=PROGRAMMATIC
//
//	→ poll until status=checkedOut (respects ctx cancellation)
//	→ GET /api/access/{transactionId}/tokens
//
// Returns ErrCheckoutTimeout (wrapped) if the context deadline is reached
// before the checkout completes.
func (c *Client) Checkout(ctx context.Context, profileID, environmentID string) (*CheckedOutProfile, *Credentials, error) {
	pollCtx, cancel := contextWithDefaultDeadline(ctx, 2*time.Minute)
	defer cancel()

	checkedOut, err := c.checkout(pollCtx, profileID, environmentID, AccessTypeProgrammatic)
	if err != nil {
		return nil, nil, err
	}
	creds, err := c.GetCredentials(pollCtx, checkedOut.TransactionID)
	if err != nil {
		return nil, nil, err
	}
	return checkedOut, creds, nil
}

// CheckoutConsole checks out a profile for console access and returns the
// federated sign-in URL for the cloud provider's web console.
// Flow: POST /api/access/{profileId}/environments/{environmentId}?accessType=CONSOLE
//
//	→ poll until status=checkedOut (respects ctx cancellation)
//	→ GET /api/access/{transactionId}/url
//
// Returns ErrCheckoutTimeout (wrapped) if the context deadline is reached
// before the checkout completes.
func (c *Client) CheckoutConsole(ctx context.Context, profileID, environmentID string) (*CheckedOutProfile, string, error) {
	pollCtx, cancel := contextWithDefaultDeadline(ctx, 2*time.Minute)
	defer cancel()

	checkedOut, err := c.checkout(pollCtx, profileID, environmentID, AccessTypeConsole)
	if err != nil {
		return nil, "", err
	}
	consoleURL, err := c.GetConsoleURL(pollCtx, checkedOut.TransactionID)
	if err != nil {
		return nil, "", err
	}
	return checkedOut, consoleURL, nil
}

// checkout initiates a checkout of the given access type and polls until
// Britive reports the transaction as checked out.
//
// Transient HTTP failures during the poll loop (connection resets, 5xx, DNS
// blips) are retried up to maxConsecutivePollFailures consecutive times
// before giving up. A single network hiccup should not abort an in-flight
// checkout, which previously caused users to see a hard failure for what was
// actually a fully-recoverable glitch.
func (c *Client) checkout(ctx context.Context, profileID, environmentID, accessType string) (*CheckedOutProfile, error) {
	if profileID == "" || environmentID == "" {
		return nil, fmt.Errorf("profileId and environmentId must not be empty")
	}

	// Initiate checkout
	var txn Transaction
	path := fmt.Sprintf("/api/access/%s/environments/%s?accessType=%s", profileID, environmentID, accessType)
	if err := c.post(ctx, path, map[string]string{}, &txn); err != nil {
		return nil, fmt.Errorf("checkout initiation failed: %w", err)
	}
	transactionID := txn.TransactionID

	// NOTE: Britive does not expose a single-transaction status endpoint, so
	// we poll MySessions (which returns every active session) and filter for
	// the one we just initiated. If the API gains a narrower endpoint, point
	// this loop at it to cut bandwidth and server-side load.
	var consecutiveFailures int
	for {
		active, err := c.MySessions(ctx)
		if err != nil {
			consecutiveFailures++
			if consecutiveFailures >= maxConsecutivePollFailures {
				return nil, fmt.Errorf("checkout polling failed after %d consecutive errors: %w", consecutiveFailures, err)
			}
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("%w: %w", ErrCheckoutTimeout, ctx.Err())
			case <-time.After(2 * time.Second):
				continue
			}
		}
		consecutiveFailures = 0

		for i := range active {
			if active[i].TransactionID == transactionID && active[i].Status == "checkedOut" {
				return &active[i], nil
			}
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w: %w", ErrCheckoutTimeout, ctx.Err())
		case <-time.After(2 * time.Second):
			// continue polling
		}
	}
}

// GetConsoleURL retrieves the console sign-in URL for an active console checkout.
// GET /api/access/{transactionId}/url
//
// Britive returns either a JSON object with a "url" field or a bare JSON
// string depending on the application type, so both shapes are accepted.
// Only absolute https URLs are returned, since the value is handed to the
// operating system's URL handler.
func (c *Client) GetConsoleURL(ctx context.Context, transactionID string) (string, error) {
	var raw json.RawMessage
	if err := c.get(ctx, fmt.Sprintf("/api/access/%s/url", transactionID), &raw); err != nil {
		return "", fmt.Errorf("fetching console URL: %w", err)
	}

	var consoleURL string
	if err := json.Unmarshal(raw, &consoleURL); err != nil {
		var obj struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return "", fmt.Errorf("decoding console URL: %w", err)
		}
		consoleURL = obj.URL
	}

	consoleURL = strings.TrimSpace(consoleURL)
	parsed, err := url.Parse(consoleURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("console URL from Britive is not a valid https URL")
	}
	return consoleURL, nil
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
