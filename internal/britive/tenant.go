package britive

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SanitizeTenant normalizes user-provided tenant input into the short
// subdomain form that bctl stores and uses to build URLs. It accepts any of:
//
//	acme
//	acme.britive-app.com
//	https://acme.britive-app.com
//	https://acme.britive-app.com/
//	  ACME  (with whitespace and case)
//
// and returns "acme" for each. Trimming prevents the double-URL bug where
// the saved value becomes https://https://acme.britive-app.com.britive-app.com.
func SanitizeTenant(input string) string {
	s := strings.TrimSpace(input)
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".britive-app.com")
	s = strings.Trim(s, "/")
	return strings.ToLower(s)
}

// CheckTenantReachable does a short-timeout HTTP HEAD against the tenant's
// base URL to confirm DNS resolves and TLS handshakes. It is intentionally
// best-effort: any successful TCP/TLS round-trip counts as reachable, even if
// the response is a 4xx, because Britive may not implement HEAD on /.
//
// Returns nil on reachable, a descriptive error otherwise. Callers should
// surface this as a warning rather than a hard failure so offline setup still
// works.
func CheckTenantReachable(ctx context.Context, tenant string) error {
	if tenant == "" {
		return fmt.Errorf("tenant is empty")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("https://%s.britive-app.com/", tenant)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("reaching %s: %w", url, err)
	}
	_ = resp.Body.Close()
	return nil
}
