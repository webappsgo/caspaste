// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package config

import (
	"github.com/casjay-forks/caspaste/src/validation"
)

// ParseBool parses a boolean value with support for 40+ truthy/falsey strings
// per AI.md PART 5: Boolean Handling specification.
// Returns: (value, wasSet)
func ParseBool(s string) (bool, bool) {
	return validation.ParseBool(s)
}
