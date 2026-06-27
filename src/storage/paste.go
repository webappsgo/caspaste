
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package storage

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// Default query timeouts per AI.md PART 10
const (
	// Simple queries (SELECT single row, INSERT, UPDATE, DELETE)
	defaultQueryTimeout = 5 * time.Second
	// List queries (SELECT multiple rows)
	defaultListTimeout = 10 * time.Second
	// Batch operations (DELETE expired, migrations)
	defaultBatchTimeout = 30 * time.Second
)

type Paste struct {
	// Ignored when creating
	ID         string `json:"id"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	// Ignored when creating
	CreateTime int64  `json:"createTime"`
	DeleteTime int64  `json:"deleteTime"`
	OneUse     bool   `json:"oneUse"`
	Syntax     string `json:"syntax"`

	Author      string `json:"author"`
	AuthorEmail string `json:"authorEmail"`
	AuthorURL   string `json:"authorURL"`

	// MicroBin-inspired features
	// True if this is a file upload
	IsFile bool `json:"isFile"`
	// Original filename for file uploads
	FileName string `json:"fileName"`
	// MIME type for file uploads
	MimeType string `json:"mimeType"`
	// Allow paste editing
	IsEditable bool `json:"isEditable"`
	// Private paste (not listed publicly)
	IsPrivate bool `json:"isPrivate"`
	// True if this is a URL shortener entry
	IsURL bool `json:"isURL"`
	// Original URL for shortener
	OriginalURL string `json:"originalURL"`
}

func (db DB) PasteAdd(paste Paste) (string, int64, int64, error) {
	var err error

	// Generate ID
	paste.ID, err = genTokenCrypto(8)
	if err != nil {
		return paste.ID, paste.CreateTime, paste.DeleteTime, err
	}

	// Set paste create time
	paste.CreateTime = time.Now().Unix()

	// Check delete time
	if paste.DeleteTime < 0 {
		paste.DeleteTime = 0
	}

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	// Add to primary database
	_, err = db.execSQL(ctx,
		`INSERT INTO pastes (id, title, body, syntax, create_time, delete_time, one_use, author, author_email, author_url, is_file, file_name, mime_type, is_editable, is_private, is_url, original_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		paste.ID, paste.Title, paste.Body, paste.Syntax, paste.CreateTime, paste.DeleteTime, paste.OneUse,
		paste.Author, paste.AuthorEmail, paste.AuthorURL,
		paste.IsFile, paste.FileName, paste.MimeType, paste.IsEditable, paste.IsPrivate, paste.IsURL, paste.OriginalURL,
	)
	if err != nil {
		return paste.ID, paste.CreateTime, paste.DeleteTime, err
	}

	// Also add to SQLite backup/cache if available
	if db.backupPool != nil {
		// Backup uses separate context
		backupCtx, backupCancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
		defer backupCancel()
		_, backupErr := db.backupPool.ExecContext(backupCtx,
			`INSERT OR REPLACE INTO pastes (id, title, body, syntax, create_time, delete_time, one_use, author, author_email, author_url, is_file, file_name, mime_type, is_editable, is_private, is_url, original_url)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			paste.ID, paste.Title, paste.Body, paste.Syntax, paste.CreateTime, paste.DeleteTime, paste.OneUse,
			paste.Author, paste.AuthorEmail, paste.AuthorURL,
			paste.IsFile, paste.FileName, paste.MimeType, paste.IsEditable, paste.IsPrivate, paste.IsURL, paste.OriginalURL,
		)
		// Log backup errors but don't fail primary operation
		// Per AI.md PART 11: warn level for recoverable issues
		if backupErr != nil {
			log.Printf("[WARN] storage: backup insert failed for paste %s: %v", paste.ID, backupErr)
		}
	}

	return paste.ID, paste.CreateTime, paste.DeleteTime, nil
}

func (db DB) PasteUpdate(paste Paste) error {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	// Update in primary database
	result, err := db.execSQL(ctx,
		`UPDATE pastes SET title = $2, body = $3, syntax = $4, delete_time = $5, one_use = $6,
		author = $7, author_email = $8, author_url = $9,
		is_file = $10, file_name = $11, mime_type = $12, is_editable = $13, is_private = $14, is_url = $15, original_url = $16
		WHERE id = $1`,
		paste.ID, paste.Title, paste.Body, paste.Syntax, paste.DeleteTime, paste.OneUse,
		paste.Author, paste.AuthorEmail, paste.AuthorURL,
		paste.IsFile, paste.FileName, paste.MimeType, paste.IsEditable, paste.IsPrivate, paste.IsURL, paste.OriginalURL,
	)
	if err != nil {
		return err
	}

	// Check result
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFoundID
	}

	// Also update in SQLite backup/cache if available
	if db.backupPool != nil {
		backupCtx, backupCancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
		defer backupCancel()
		_, backupErr := db.backupPool.ExecContext(backupCtx,
			`UPDATE pastes SET title = ?, body = ?, syntax = ?, delete_time = ?, one_use = ?,
			author = ?, author_email = ?, author_url = ?,
			is_file = ?, file_name = ?, mime_type = ?, is_editable = ?, is_private = ?, is_url = ?, original_url = ?
			WHERE id = ?`,
			paste.Title, paste.Body, paste.Syntax, paste.DeleteTime, paste.OneUse,
			paste.Author, paste.AuthorEmail, paste.AuthorURL,
			paste.IsFile, paste.FileName, paste.MimeType, paste.IsEditable, paste.IsPrivate, paste.IsURL, paste.OriginalURL,
			paste.ID,
		)
		// Log backup errors but don't fail primary operation
		if backupErr != nil {
			log.Printf("[WARN] storage: backup update failed for paste %s: %v", paste.ID, backupErr)
		}
	}

	return nil
}

func (db DB) PasteDelete(id string) error {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	// Delete from primary database
	result, err := db.execSQL(ctx,
		`DELETE FROM pastes WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	// Check result
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFoundID
	}

	// Also delete from SQLite backup/cache if available
	if db.backupPool != nil {
		backupCtx, backupCancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
		defer backupCancel()
		_, backupErr := db.backupPool.ExecContext(backupCtx, `DELETE FROM pastes WHERE id = ?`, id)
		// Log backup errors but don't fail primary operation
		if backupErr != nil {
			log.Printf("[WARN] storage: backup delete failed for paste %s: %v", id, backupErr)
		}
	}

	return nil
}

// PasteDeleteIfOneUse atomically deletes a one-use paste.
// Returns (true, nil) if this caller successfully deleted it.
// Returns (false, nil) if another concurrent request already consumed it.
func (db DB) PasteDeleteIfOneUse(id string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	result, err := db.execSQL(ctx,
		`DELETE FROM pastes WHERE id = $1 AND one_use = 1`,
		id,
	)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	if rowsAffected > 0 && db.backupPool != nil {
		backupCtx, backupCancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
		defer backupCancel()
		_, _ = db.backupPool.ExecContext(backupCtx, `DELETE FROM pastes WHERE id = ?`, id)
	}

	return rowsAffected > 0, nil
}

func (db DB) PasteGet(id string) (Paste, error) {
	var paste Paste

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	// Make query
	row := db.queryRowSQL(ctx,
		`SELECT id, title, body, syntax, create_time, delete_time, one_use, author, author_email, author_url,
		is_file, file_name, mime_type, is_editable, is_private, is_url, original_url
		FROM pastes WHERE id = $1`,
		id,
	)

	// Read query
	err := row.Scan(&paste.ID, &paste.Title, &paste.Body, &paste.Syntax, &paste.CreateTime, &paste.DeleteTime, &paste.OneUse,
		&paste.Author, &paste.AuthorEmail, &paste.AuthorURL,
		&paste.IsFile, &paste.FileName, &paste.MimeType, &paste.IsEditable, &paste.IsPrivate, &paste.IsURL, &paste.OriginalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return paste, ErrNotFoundID
		}

		return paste, err
	}

	// Check paste expiration
	if paste.DeleteTime < time.Now().Unix() && paste.DeleteTime > 0 {
		// Delete expired paste with timeout
		delCtx, delCancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
		defer delCancel()
		_, err = db.execSQL(delCtx,
			`DELETE FROM pastes WHERE id = $1`,
			paste.ID,
		)
		if err != nil {
			return Paste{}, err
		}

		// Return ErrNotFound
		return Paste{}, ErrNotFoundID
	}

	return paste, nil
}

func (db DB) PasteDeleteExpired() (int64, error) {
	// Batch timeout per AI.md PART 10 (longer for batch operations)
	ctx, cancel := context.WithTimeout(context.Background(), defaultBatchTimeout)
	defer cancel()

	// Delete from primary database
	result, err := db.execSQL(ctx,
		`DELETE FROM pastes WHERE (delete_time < $1) AND (delete_time > 0)`,
		time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}

	// Check result
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return rowsAffected, err
	}

	// Also delete from SQLite backup/cache if available
	if db.backupPool != nil {
		backupCtx, backupCancel := context.WithTimeout(context.Background(), defaultBatchTimeout)
		defer backupCancel()
		_, backupErr := db.backupPool.ExecContext(backupCtx,
			`DELETE FROM pastes WHERE (delete_time < ?) AND (delete_time > 0)`,
			time.Now().Unix(),
		)
		// Log backup errors but don't fail primary operation
		if backupErr != nil {
			log.Printf("[WARN] storage: backup delete expired failed: %v", backupErr)
		}
	}

	return rowsAffected, nil
}

type PasteListItem struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Syntax     string `json:"syntax"`
	CreateTime int64  `json:"createTime"`
	DeleteTime int64  `json:"deleteTime"`
}

func (db DB) PasteList(limit int, offset int) ([]PasteListItem, error) {
	if limit <= 0 || limit > 100 {
		// Default limit
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// List timeout per AI.md PART 10 (longer for list queries)
	ctx, cancel := context.WithTimeout(context.Background(), defaultListTimeout)
	defer cancel()

	// Query pastes (exclude expired, one-use, and private pastes)
	rows, err := db.querySQL(ctx,
		`SELECT id, title, syntax, create_time, delete_time
		FROM pastes
		WHERE (delete_time > $1 OR delete_time = 0)
		AND is_private = FALSE
		ORDER BY create_time DESC
		LIMIT $2 OFFSET $3`,
		time.Now().Unix(),
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pastes []PasteListItem
	for rows.Next() {
		var paste PasteListItem
		err := rows.Scan(&paste.ID, &paste.Title, &paste.Syntax, &paste.CreateTime, &paste.DeleteTime)
		if err != nil {
			return nil, err
		}
		pastes = append(pastes, paste)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return pastes, nil
}
