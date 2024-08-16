package gtfs2sqlite

import (
	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestImportsValid(t *testing.T) {
	outDir := testTempdir(t)
	_, err := Import("./sample_data/sample-feed.zip", outDir+"/feed.db", nil)
	require.NoError(t, err)
}

func TestImportsEmptyAsNull(t *testing.T) {
	outDir := testTempdir(t)
	_, err := Import("./sample_data/sample-feed.zip", outDir+"/feed.db", nil)

	conn, err := sqlite.OpenConn(outDir+"/feed.db", sqlite.SQLITE_OPEN_READONLY)
	require.NoError(t, err)

	var count int
	err = sqlitex.Exec(conn, "SELECT count(*) as count FROM routes WHERE route_color IS NULL", func(stmt *sqlite.Stmt) error {
		count = int(stmt.GetInt64("count"))
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, 5, count)
}

func TestImportInvalid(t *testing.T) {
	inputs := []string{"invalid-foreign-key.zip"}
	for _, input := range inputs {
		t.Run(input+"/nofix", func(t *testing.T) {
			outDir := testTempdir(t)
			issues, err := Import("./sample_data/"+input, outDir+"/imported.db", nil)
			require.ErrorIs(t, err, ErrInvalidInput)
			require.Len(t, issues, 1)
		})
		t.Run(input+"/ignore", func(t *testing.T) {
			outDir := testTempdir(t)
			issues, err := Import("./sample_data/"+input, outDir+"/imported.db", &ImportOpts{IgnoreInvalid: true})
			require.NoError(t, err)
			require.Len(t, issues, 1)
		})
		t.Run(input+"/fix", func(t *testing.T) {
			outDir := testTempdir(t)
			issues, err := Import("./sample_data/"+input, outDir+"/imported.db", &ImportOpts{ForceValid: true})
			require.NoError(t, err)
			require.Len(t, issues, 1)
		})
	}
}

func testTempdir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	t.Cleanup(func() {
		if t.Failed() {
			fmt.Println("Preserving tempdir after failed test", dir)
		} else {
			_ = os.RemoveAll(dir)
		}
	})
	return dir
}
