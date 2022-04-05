package util

import "regexp"

var branchRegex = regexp.MustCompilePOSIX(`[a-zA-Z0-9_\-]+`)

// ValidateTargetBranchName ensures that the prefix matches a pattern of only strings, numbers, and dashes.
func ValidateTargetBranchName(name string) bool {
	return branchRegex.MatchString(name)
}

// ValidateVirtualBranchPrefix ensures that the prefix matches a pattern of only strings, numbers, and dashes.
func ValidateVirtualBranchPrefix(name string) bool {
	return branchRegex.MatchString(name)
}

// StringLookup is an alias map that maps a string to a bool that is always true if the key exists.
type StringLookup map[string]bool
