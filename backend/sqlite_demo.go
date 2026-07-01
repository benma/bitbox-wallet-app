// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/BitBoxSwiss/bitbox-wallet-app/util/errp"
	_ "github.com/mattn/go-sqlite3"
)

const sqliteDemoKey = "bitbox-sqlcipher-demo-key"

type sqliteDemoRow struct {
	ID    int64
	Title string
}

type sqliteDemoResult struct {
	CipherVersion string
	Rows          []sqliteDemoRow
}

func runSQLiteDemo() (*sqliteDemoResult, error) {
	file, err := os.CreateTemp("", "bitbox-sqlcipher-demo-*.db")
	if err != nil {
		return nil, errp.WithStack(err)
	}
	dbFilename := file.Name()
	if err := file.Close(); err != nil {
		os.Remove(dbFilename)
		return nil, errp.WithStack(err)
	}
	defer os.Remove(dbFilename)

	db, err := sql.Open("sqlite3", sqliteDemoDSN(dbFilename, sqliteDemoKey))
	if err != nil {
		return nil, errp.WithStack(err)
	}
	db.SetMaxOpenConns(1)

	cipherVersion, err := sqliteDemoCipherVersion(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	if _, err := db.Exec(`
		CREATE TABLE demo_rows (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL
		)
	`); err != nil {
		db.Close()
		return nil, errp.WithStack(err)
	}

	tx, err := db.Begin()
	if err != nil {
		db.Close()
		return nil, errp.WithStack(err)
	}
	stmt, err := tx.Prepare("INSERT INTO demo_rows (title) VALUES (?)")
	if err != nil {
		tx.Rollback()
		db.Close()
		return nil, errp.WithStack(err)
	}

	for _, title := range []string{"created table", "wrote rows", "read rows"} {
		if _, err := stmt.Exec(title); err != nil {
			tx.Rollback()
			stmt.Close()
			db.Close()
			return nil, errp.WithStack(err)
		}
	}
	if err := stmt.Close(); err != nil {
		tx.Rollback()
		db.Close()
		return nil, errp.WithStack(err)
	}
	if err := tx.Commit(); err != nil {
		db.Close()
		return nil, errp.WithStack(err)
	}
	if err := db.Close(); err != nil {
		return nil, errp.WithStack(err)
	}

	if err := sqliteDemoAssertEncrypted(dbFilename); err != nil {
		return nil, err
	}
	if err := sqliteDemoAssertCannotRead(dbFilename, ""); err != nil {
		return nil, err
	}
	if err := sqliteDemoAssertCannotRead(dbFilename, "wrong-key"); err != nil {
		return nil, err
	}

	db, err = sql.Open("sqlite3", sqliteDemoDSN(dbFilename, sqliteDemoKey))
	if err != nil {
		return nil, errp.WithStack(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	rows, err := db.Query("SELECT id, title FROM demo_rows ORDER BY id")
	if err != nil {
		return nil, errp.WithStack(err)
	}
	defer rows.Close()

	var result []sqliteDemoRow
	for rows.Next() {
		var row sqliteDemoRow
		if err := rows.Scan(&row.ID, &row.Title); err != nil {
			return nil, errp.WithStack(err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, errp.WithStack(err)
	}
	if len(result) != 3 {
		return nil, errp.WithStack(fmt.Errorf("unexpected sqlite demo row count: %d", len(result)))
	}

	return &sqliteDemoResult{
		CipherVersion: cipherVersion,
		Rows:          result,
	}, nil
}

func sqliteDemoDSN(filename string, key string) string {
	u := url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(filename),
	}
	if key != "" {
		query := u.Query()
		query.Set("_key", key)
		u.RawQuery = query.Encode()
	}
	return u.String()
}

func sqliteDemoCipherVersion(db *sql.DB) (string, error) {
	var version string
	if err := db.QueryRow("PRAGMA cipher_version;").Scan(&version); err != nil {
		return "", errp.WithStack(fmt.Errorf(
			"SQLCipher is not available; build with the sqlcipher or libsqlcipher tag: %w", err))
	}
	if version == "" {
		return "", errp.WithStack(fmt.Errorf("empty SQLCipher version"))
	}
	return version, nil
}

func sqliteDemoAssertEncrypted(filename string) error {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return errp.WithStack(err)
	}
	if bytes.HasPrefix(contents, []byte("SQLite format 3\x00")) {
		return errp.WithStack(fmt.Errorf("SQLCipher demo database has a plaintext SQLite header"))
	}
	return nil
}

func sqliteDemoAssertCannotRead(filename string, key string) error {
	db, err := sql.Open("sqlite3", sqliteDemoDSN(filename, key))
	if err != nil {
		return nil
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM demo_rows").Scan(&count); err == nil {
		return errp.WithStack(fmt.Errorf("SQLCipher demo database was readable with key %q", key))
	}
	return nil
}
