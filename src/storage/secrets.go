
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"
)

const secretsTimeout = 5 * time.Second

// LoadOrGenerateSecret returns the persisted value of a named secret from the
// app_secrets table. If the row does not exist, a fresh 32-byte random secret
// is generated, persisted, and returned. This guarantees that secrets like
// cookie_signing_key survive server restarts (per AI.md PART 11).
//
// The returned byte slice is the decoded raw bytes (32 bytes / 256 bits).
func (db DB) LoadOrGenerateSecret(name string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), secretsTimeout)
	defer cancel()

	var encoded string
	err := db.pool.QueryRowContext(ctx,
		`SELECT value FROM app_secrets WHERE name = ?`, name,
	).Scan(&encoded)

	if err == nil {
		raw, decErr := hex.DecodeString(encoded)
		if decErr != nil {
			return nil, decErr
		}
		return raw, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Generate a new 32-byte secret
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	encoded = hex.EncodeToString(raw)

	_, err = db.pool.ExecContext(ctx,
		`INSERT INTO app_secrets (name, value) VALUES (?, ?)`, name, encoded,
	)
	if err != nil {
		return nil, err
	}
	return raw, nil
}
