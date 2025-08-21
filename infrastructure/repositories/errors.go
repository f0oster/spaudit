package repositories

import "fmt"

// ErrSiteMismatch occurs when a domain object's site ID doesn't match the repository's scope
type ErrSiteMismatch struct {
	Expected int64
	Actual   int64
}

func (e ErrSiteMismatch) Error() string {
	return fmt.Sprintf("site ID mismatch: repository scoped to site %d, but object has site ID %d", e.Expected, e.Actual)
}
