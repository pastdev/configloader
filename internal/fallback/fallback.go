// Package fallback provides helpers shared by template func collections that
// accept an optional fallback value.
package fallback

// First returns the first fallback value and true if one was supplied and is
// non-empty.
func First(fallback ...string) (string, bool) {
	if len(fallback) == 0 || fallback[0] == "" {
		return "", false
	}
	return fallback[0], true
}
