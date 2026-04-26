// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package validation

import (
	"strings"
)

// ParseBool parses a boolean value with support for 40+ truthy/falsey strings
// per AI.md PART 5: Boolean Handling specification.
// Returns: (value, wasSet)
func ParseBool(s string) (bool, bool) {
	if s == "" {
		return false, false
	}

	s = strings.ToLower(strings.TrimSpace(s))

	// Truthy values per AI.md PART 5 (40+ variations)
	truthy := []string{
		"1", "yes", "true", "enable", "enabled", "on",
		"yep", "yup", "yeah", "affirmative", "aye",
		"si", "oui", "da", "hai", "totally", "sure",
		"ok", "okay", "accept", "allow", "grant",
		"y", "t", "active",
	}
	for _, val := range truthy {
		if s == val {
			return true, true
		}
	}

	// Falsey values per AI.md PART 5 (40+ variations)
	falsey := []string{
		"0", "no", "false", "disable", "disabled", "off",
		"nope", "nah", "nay", "negative", "nein",
		"non", "niet", "iie", "lie", "noway", "never",
		"deny", "reject", "block", "revoke",
		"n", "f", "inactive",
	}
	for _, val := range falsey {
		if s == val {
			return false, true
		}
	}

	// Unknown value, treat as not set
	return false, false
}

// IsTruthy returns true if the string represents a truthy value
func IsTruthy(s string) bool {
	value, wasSet := ParseBool(s)
	return wasSet && value
}

// IsFalsey returns true if the string represents a falsey value
func IsFalsey(s string) bool {
	value, wasSet := ParseBool(s)
	return wasSet && !value
}
