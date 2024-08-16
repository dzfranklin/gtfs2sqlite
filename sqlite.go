package gtfs2sqlite

import "crawshaw.io/sqlite"

func sqlitexNoop(stmt *sqlite.Stmt) error {
	return stmt.Finalize()
}
