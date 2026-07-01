// SPDX-License-Identifier: Apache-2.0

//go:build sqlcipher || libsqlcipher
// +build sqlcipher libsqlcipher

package backend

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunSQLiteDemo(t *testing.T) {
	result, err := runSQLiteDemo()
	require.NoError(t, err)
	require.NotEmpty(t, result.CipherVersion)
	require.Equal(t, []sqliteDemoRow{
		{ID: 1, Title: "created table"},
		{ID: 2, Title: "wrote rows"},
		{ID: 3, Title: "read rows"},
	}, result.Rows)
}
