// SPDX-License-Identifier: Apache-2.0

//go:build sqlcipher || libsqlcipher
// +build sqlcipher libsqlcipher

package backend

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunSQLiteDemo(t *testing.T) {
	result, err := runSQLiteDemo(t.TempDir())
	require.NoError(t, err)
	require.NotEmpty(t, result.CipherVersion)
	require.Equal(t, []SQLiteDemoRow{
		{ID: 1, Title: "created table"},
		{ID: 2, Title: "wrote rows"},
		{ID: 3, Title: "read rows"},
	}, result.Rows)
}

func TestSQLiteDemoDSNWindowsDrivePath(t *testing.T) {
	filename := `C:\Users\alice\AppData\Roaming\BitBox\demo.db`
	key := "x'000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f'"
	dsn := sqliteDemoDSN(filename, key)

	dsnFilename, rawQuery, ok := strings.Cut(dsn, "?")
	require.True(t, ok)
	require.Equal(t, filename, dsnFilename)
	require.NotContains(t, dsn, "file:")

	query, err := url.ParseQuery(rawQuery)
	require.NoError(t, err)
	require.Equal(t, key, query.Get("_key"))
}
