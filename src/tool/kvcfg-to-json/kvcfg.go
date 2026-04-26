
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package main

import (
	"errors"
	"strconv"
	"strings"
)

func readKVCfg(data string) (map[string]string, error) {
	out := make(map[string]string)

	dataSplit := strings.Split(data, "\n")
	dataSplitLen := len(dataSplit)

	for num := 0; num < dataSplitLen; num++ {
		str := strings.TrimSpace(dataSplit[num])

		if str == "" || strings.HasPrefix(str, "//") {
			continue
		}

		strSplit := strings.SplitN(str, "=", 2)
		if len(strSplit) != 2 {
			return out, errors.New("error in line " + strconv.Itoa(num+1) + ": expected '=' delimiter")
		}

		key := strings.TrimSpace(strSplit[0])
		val := strings.TrimSpace(strSplit[1])
		val, isMultiline := multilineCheck(val)

		if isMultiline {
			num = num + 1
			for ; num < dataSplitLen; num++ {
				strPlus := strings.TrimSpace(dataSplit[num])
				strPlus, isMultilinePlus := multilineCheck(strPlus)
				val = val + strPlus

				if !isMultilinePlus {
					break
				}
			}
		}

		_, exist := out[key]
		if exist {
			return out, errors.New("duplicate key: " + key)
		}

		out[key] = val
	}

	return out, nil
}

func multilineCheck(s string) (string, bool) {
	sLen := len(s)

	if sLen > 0 && s[sLen-1] == '\\' {
		if sLen > 1 && s[sLen-2] == '\\' {
			return s[:sLen-1], false
		}

		return s[:sLen-1], true
	}

	return s, false
}
