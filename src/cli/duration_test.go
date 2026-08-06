// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package cli

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	testData := map[string]time.Duration{
		"10m":   60 * 10 * time.Second,
		"1h 1d": 60 * 60 * 25 * time.Second,
		"1h1d":  60 * 60 * 25 * time.Second,
		"1w":    60 * 60 * 24 * 7 * time.Second,
		"365d":  60 * 60 * 24 * 365 * time.Second,
	}

	for s, exp := range testData {
		res, err := ParseDuration(s)
		if err != nil {
			t.Fatal(err)
		}

		if exp != res {
			t.Error("expected", exp, "but got", res, "(input:", s, ")")
		}
	}
}
