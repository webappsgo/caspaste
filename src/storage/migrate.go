
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package storage

import (
	"context"
	"fmt"
	"time"
)

// Migration timeout - longer for batch operations
const migrationTimeout = 5 * time.Minute

// MigrateDatabase migrates all data from source database to destination database
func MigrateDatabase(sourceDriver, sourceSource, destDriver, destSource string) error {
	fmt.Println("Database Migration")
	fmt.Println("==================")
	fmt.Printf("Source: %s (%s)\n", sourceDriver, sourceSource)
	fmt.Printf("Destination: %s (%s)\n", destDriver, destSource)
	fmt.Println()

	// Open source database
	fmt.Println("Opening source database...")
	sourceDB, err := NewPool(sourceDriver, sourceSource, 25, 5, "")
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer sourceDB.Close()

	// Open destination database
	fmt.Println("Opening destination database...")
	destDB, err := NewPool(destDriver, destSource, 25, 5, "")
	if err != nil {
		return fmt.Errorf("failed to open destination database: %w", err)
	}
	defer destDB.Close()

	// Initialize destination database schema
	fmt.Println("Initializing destination schema...")
	err = InitDB(destDriver, destSource)
	if err != nil {
		return fmt.Errorf("failed to initialize destination schema: %w", err)
	}

	// Migration timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	// Query all pastes from source
	fmt.Println("Reading pastes from source database...")
	rows, err := sourceDB.pool.QueryContext(ctx, `
		SELECT id, title, body, syntax, create_time, delete_time, one_use,
		       author, author_email, author_url,
		       COALESCE(is_file, 0), COALESCE(file_name, ''), COALESCE(mime_type, ''),
		       COALESCE(is_editable, 0), COALESCE(is_private, 0),
		       COALESCE(is_url, 0), COALESCE(original_url, '')
		FROM pastes
	`)
	if err != nil {
		return fmt.Errorf("failed to read source database: %w", err)
	}
	defer rows.Close()

	// Migrate each paste
	count := 0
	fmt.Println("Migrating pastes...")
	for rows.Next() {
		var paste Paste
		err := rows.Scan(
			&paste.ID, &paste.Title, &paste.Body, &paste.Syntax,
			&paste.CreateTime, &paste.DeleteTime, &paste.OneUse,
			&paste.Author, &paste.AuthorEmail, &paste.AuthorURL,
			&paste.IsFile, &paste.FileName, &paste.MimeType,
			&paste.IsEditable, &paste.IsPrivate, &paste.IsURL, &paste.OriginalURL,
		)
		if err != nil {
			return fmt.Errorf("failed to scan paste: %w", err)
		}

		// Insert into destination database with timeout
		insertCtx, insertCancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
		_, err = destDB.execSQL(insertCtx, `
			INSERT INTO pastes (id, title, body, syntax, create_time, delete_time, one_use,
			                    author, author_email, author_url,
			                    is_file, file_name, mime_type, is_editable, is_private, is_url, original_url)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		`, paste.ID, paste.Title, paste.Body, paste.Syntax,
			paste.CreateTime, paste.DeleteTime, paste.OneUse,
			paste.Author, paste.AuthorEmail, paste.AuthorURL,
			paste.IsFile, paste.FileName, paste.MimeType,
			paste.IsEditable, paste.IsPrivate, paste.IsURL, paste.OriginalURL)
		insertCancel()

		if err != nil {
			return fmt.Errorf("failed to insert paste %s: %w", paste.ID, err)
		}

		count++
		if count%100 == 0 {
			fmt.Printf("Migrated %d pastes...\n", count)
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error reading source rows: %w", err)
	}

	fmt.Println()
	fmt.Printf("Migration complete! Migrated %d pastes.\n", count)
	fmt.Println()
	fmt.Println("IMPORTANT: Verify the migration before switching to the new database!")
	fmt.Printf("  1. Check destination database: %s\n", destSource)
	fmt.Printf("  2. Test a few paste IDs\n")
	fmt.Printf("  3. Update your configuration to use the new database\n")
	fmt.Println()

	return nil
}
