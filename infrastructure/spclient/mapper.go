package spclient

import (
	"net/url"
	"strings"
)

// joinURL safely joins a base URL with a relative path
func joinURL(base, rel string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	if strings.HasPrefix(rel, "/") {
		u.Path = rel
		return u.String()
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	u.Path += rel
	return u.String()
}

// firstNonEmpty returns the first non-empty string from the provided values
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
