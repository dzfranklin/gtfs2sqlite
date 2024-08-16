package gtfs2sqlite

import (
	"archive/zip"
	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"
)

type ExportOpts struct{}

func Export(inputPath string, outputPath string, opts *ExportOpts) error {
	if inputPath == "" {
		panic("Missing inputPath")
	}
	if outputPath == "" {
		panic("Missing outputPath")
	}

	slog.Info(fmt.Sprintf("Exporting %s to %s", inputPath, outputPath))

	db, err := sqlite.OpenConn(inputPath, sqlite.SQLITE_OPEN_READONLY)
	if err != nil {
		return err
	}
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()

	outputF, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	outputZip := zip.NewWriter(outputF)
	defer func() {
		_ = outputZip.Close()
		_ = outputF.Close()
	}()

	var tables []string
	err = sqlitex.Exec(db, "SELECT name FROM sqlite_master WHERE type = 'table'", func(stmt *sqlite.Stmt) error {
		tables = append(tables, stmt.GetText("name"))
		return nil
	})
	if err != nil {
		return err
	}

	if slices.Contains(tables, "__gtfs2sqlite_other_files") {
		if err := exportOtherFiles(db, outputZip); err != nil {
			return err
		}
	}

	var emptyFileTables []string
	if slices.Contains(tables, "__gtfs2sqlite_empty_files") {
		err = sqlitex.Exec(db, "SELECT tableName FROM __gtfs2sqlite_empty_files", func(stmt *sqlite.Stmt) error {
			emptyFileTables = append(emptyFileTables, stmt.GetText("tableName"))
			return nil
		})
		if err != nil {
			return err
		}
	}

	for _, table := range tables {
		if strings.HasPrefix(table, "__gtfs2sqlite") {
			continue
		}

		var rowCount int64
		err = sqlitex.Exec(db, fmt.Sprintf("SELECT count(*) AS count FROM %s", table), func(stmt *sqlite.Stmt) error {
			rowCount = stmt.GetInt64("count")
			return nil
		})
		if err != nil {
			return err
		}
		if rowCount == 0 && !slices.Contains(emptyFileTables, table) {
			continue
		}

		if err := exportTableIn(db, outputZip, table); err != nil {
			return err
		}
	}

	if err := outputZip.Close(); err != nil {
		return err
	}
	if err := outputF.Close(); err != nil {
		return err
	}

	err = db.Close()
	db = nil
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Wrote %s", outputPath))
	return nil
}

func exportTableIn(db *sqlite.Conn, outputZip *zip.Writer, table string) error {
	outputName := table + ".txt"
	outputF, err := outputZip.Create(outputName)
	if err != nil {
		return err
	}
	outputCSV := csv.NewWriter(outputF)
	defer func() {
		outputCSV.Flush()
	}()

	rowCount := 0

	var cols []string
	err = sqlitex.Exec(db, "SELECT name FROM pragma_table_info(?)", func(stmt *sqlite.Stmt) error {
		cols = append(cols, stmt.GetText("name"))
		return nil
	}, table)
	if err != nil {
		return err
	}
	if err := outputCSV.Write(cols); err != nil {
		return err
	}
	rowCount++

	err = sqlitex.Exec(db, "SELECT * FROM "+table, func(stmt *sqlite.Stmt) error {
		var row []string
		for _, col := range cols {
			row = append(row, stmt.GetText(col))
		}
		if err := outputCSV.Write(row); err != nil {
			return err
		}
		rowCount++
		return nil
	})
	if err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Wrote %d rows to %s", rowCount, outputName))

	outputCSV.Flush()
	return outputCSV.Error()
}

func exportOtherFiles(db *sqlite.Conn, outputZip *zip.Writer) error {
	err := sqlitex.ExecTransient(db, "SELECT name, contents FROM __gtfs2sqlite_other_files", func(stmt *sqlite.Stmt) error {
		name := stmt.GetText("name")
		contents := stmt.GetReader("contents")

		outputF, err := outputZip.Create(name)
		if err != nil {
			return err
		}

		byteLen, err := io.Copy(outputF, contents)
		if err != nil {
			return err
		}
		slog.Info(fmt.Sprintf("Exported other file %s (%d bytes)", name, byteLen))
		return nil
	})
	return err
}
