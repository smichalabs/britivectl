package britive

import "errors"

// Sentinel errors returned by the Britive client and auth package.
// Callers can use errors.Is to check for these conditions.
var (
	// ErrNotLoggedIn indicates the user has no stored token for the tenant.
	ErrNotLoggedIn = errors.New("not logged in")

	// ErrUnauthorized indicates the Britive API rejected the credentials (HTTP 401).
	ErrUnauthorized = errors.New("unauthorized")

	// ErrTokenExpired indicates a JWT whose exp claim is in the past.
	ErrTokenExpired = errors.New("token expired")

	// ErrCheckoutTimeout is returned when a checkout does not reach checkedOut
	// status before the context deadline.
	ErrCheckoutTimeout = errors.New("checkout timed out")

	// ErrAuthTimeout is returned when browser-based auth does not complete
	// before the context deadline.
	ErrAuthTimeout = errors.New("authentication timed out")

	// ErrUnsupportedPlatform indicates bctl cannot open a browser on the current OS.
	ErrUnsupportedPlatform = errors.New("unsupported platform")

	// ErrProfileNotFound indicates a profile alias was not found in the local config.
	ErrProfileNotFound = errors.New("profile not found")
)
