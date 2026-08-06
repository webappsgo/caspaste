// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package storage

import (
	"crypto/rand"
	"math/big"
)

func genTokenCrypto(tokenLen int) (string, error) {
	// Generate token
	var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	charsLen := int64(len(chars))
	charsLenBig := big.NewInt(charsLen)

	token := ""

	for i := 0; i < tokenLen; i++ {
		randInt, err := rand.Int(rand.Reader, charsLenBig)
		if err != nil {
			return "", err
		}

		token = token + string(chars[randInt.Int64()])
	}

	return token, nil
}
