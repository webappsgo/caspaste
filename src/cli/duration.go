// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package cli

import (
	"errors"
	"strconv"
	"time"
)

func ParseDuration(s string) (time.Duration, error) {
	var out int64

	var tmp string
	for _, c := range s {
		if c == ' ' {
			continue
		}

		if '0' <= c && c <= '9' {
			tmp += string(c)
			continue
		}

		val, err := strconv.ParseInt(tmp, 10, 64)
		if err != nil {
			return 0, errors.New("invalid format \"" + s + "\"")
		}

		switch c {
		case 'm':
			out += val * 60
		case 'h':
			out += val * 60 * 60
		case 'd':
			out += val * 60 * 60 * 24
		case 'w':
			out += val * 60 * 60 * 24 * 7
		default:
			return 0, errors.New("invalid format \"" + s + "\"")
		}

		tmp = ""
	}

	return time.Duration(out) * time.Second, nil
}
