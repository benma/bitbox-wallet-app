// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	utilConfig "github.com/BitBoxSwiss/bitbox-wallet-app/util/config"
	"github.com/BitBoxSwiss/bitbox-wallet-app/util/errp"
	_ "github.com/mattn/go-sqlite3"
)

const sqliteDemoRawKeySize = 32

type SQLiteDemoRow struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type SQLiteDemoResult struct {
	CipherVersion string
	Rows          []SQLiteDemoRow
}

func RunSQLiteDemo() (*SQLiteDemoResult, error) {
	return runSQLiteDemo(utilConfig.AppDir())
}

func runSQLiteDemo(baseDir string) (*SQLiteDemoResult, error) {
	rawKey, err := sqliteDemoRawKey()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, errp.WithStack(err)
	}
	demoDir, err := os.MkdirTemp(baseDir, "bitbox-sqlcipher-demo-")
	if err != nil {
		return nil, errp.WithStack(err)
	}
	defer os.RemoveAll(demoDir)
	dbFilename := filepath.Join(demoDir, "demo.db")

	db, err := sql.Open("sqlite3", sqliteDemoDSN(dbFilename, rawKey))
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

	db, err = sql.Open("sqlite3", sqliteDemoDSN(dbFilename, rawKey))
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

	var result []SQLiteDemoRow
	for rows.Next() {
		var row SQLiteDemoRow
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

	return &SQLiteDemoResult{
		CipherVersion: cipherVersion,
		Rows:          result,
	}, nil
}

func sqliteDemoRawKey() (string, error) {
	var key [sqliteDemoRawKeySize]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", errp.WithStack(err)
	}
	return "x'" + hex.EncodeToString(key[:]) + "'", nil
}

func sqliteDemoDSN(filename string, key string) string {
	if key == "" {
		return filename
	}
	query := url.Values{}
	query.Set("_key", key)
	return filename + "?" + query.Encode()
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
