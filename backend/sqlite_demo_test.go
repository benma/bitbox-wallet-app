// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunSQLiteDemo(t *testing.T) {
	rows, err := runSQLiteDemo()
	require.NoError(t, err)
	require.Equal(t, []sqliteDemoRow{
		{ID: 1, Title: "created table"},
		{ID: 2, Title: "wrote rows"},
		{ID: 3, Title: "read rows"},
	}, rows)
}
