package contracts

import "errors"

// Common errors for domain contracts
var (
	// ErrSiteScopeMismatch occurs when a repository scoped to one site ID receives a request for a different site ID
	ErrSiteScopeMismatch = errors.New("repository scoped to different site ID")
)