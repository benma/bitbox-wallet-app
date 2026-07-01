// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"database/sql"
	"fmt"

	"github.com/BitBoxSwiss/bitbox-wallet-app/util/errp"
	_ "github.com/mattn/go-sqlite3"
)

type sqliteDemoRow struct {
	ID    int64
	Title string
}

func runSQLiteDemo() ([]sqliteDemoRow, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, errp.WithStack(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`
		CREATE TABLE demo_rows (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL
		)
	`); err != nil {
		return nil, errp.WithStack(err)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, errp.WithStack(err)
	}
	stmt, err := tx.Prepare("INSERT INTO demo_rows (title) VALUES (?)")
	if err != nil {
		tx.Rollback()
		return nil, errp.WithStack(err)
	}
	defer stmt.Close()

	for _, title := range []string{"created table", "wrote rows", "read rows"} {
		if _, err := stmt.Exec(title); err != nil {
			tx.Rollback()
			return nil, errp.WithStack(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, errp.WithStack(err)
	}

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

	return result, nil
}
