package gtfs2sqlite

import (
	"archive/zip"
	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type ImportOpts struct {
	ForceValid    bool
	IgnoreInvalid bool
}

var importPragmas = map[string]string{
	"synchronous": "OFF",
}

func Import(inputPath string, outputPath string, opts *ImportOpts) ([]string, error) {
	if inputPath == "" {
		panic("Missing inputPath")
	}
	if outputPath == "" {
		panic("Missing outputPath")
	}

	if opts == nil {
		opts = &ImportOpts{}
	}

	slog.Info(fmt.Sprintf("Importing %s to %s", inputPath, outputPath))

	inputZip, err := zip.OpenReader(inputPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = inputZip.Close() }()

	err = os.Remove(outputPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	db, err := sqlite.OpenConn(outputPath, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()

	for pragma, value := range importPragmas {
		err = sqlitex.Exec(db, "PRAGMA "+pragma+" = "+value, sqlitexNoop)
		if err != nil {
			return nil, err
		}
	}

	for table, schema := range gtfsSchema {
		if err := createTable(db, table, schema); err != nil {
			return nil, err
		}
	}

	for _, filename := range inputZip.File {
		err = importFileIn(inputZip, db, filename.Name)
		if err != nil {
			return nil, err
		}
	}

	var validationLogLevel slog.Level
	if opts.ForceValid || opts.IgnoreInvalid {
		validationLogLevel = slog.LevelWarn
	} else {
		validationLogLevel = slog.LevelError
	}

	validationErrors, err := validate(db, validateOpts{
		force:    opts.ForceValid,
		ignore:   opts.IgnoreInvalid,
		logLevel: validationLogLevel,
	})
	if err != nil {
		return validationErrors, err
	}

	err = db.Close()
	db = nil
	if err != nil {
		return nil, err
	}

	slog.Info(fmt.Sprintf("Wrote %s", outputPath))
	return validationErrors, nil
}

func createTable(db *sqlite.Conn, table string, schema tableSchema) error {
	var columnFragments []string
	for column, _ := range schema.Columns {
		columnFragments = append(columnFragments, column+" TEXT")
	}
	query := fmt.Sprintf("CREATE TABLE %s (%s)", table, strings.Join(columnFragments, ", "))
	return sqlitex.ExecTransient(db, query, sqlitexNoop)
}

func importFileIn(inputZip *zip.ReadCloser, db *sqlite.Conn, filename string) error {
	inputF, err := inputZip.Open(filename)
	if err != nil {
		return err
	}
	defer func() { _ = inputF.Close() }()

	if !strings.HasSuffix(filename, ".txt") {
		slog.Info("Importing other file " + filename)

		contents, err := io.ReadAll(inputF)
		if err != nil {
			return err
		}

		if err := sqlitex.Exec(db, "CREATE TABLE IF NOT EXISTS __gtfs2sqlite_other_files (name TEXT, contents BLOB)", sqlitexNoop); err != nil {
			return err
		}
		if err := sqlitex.Exec(db, "INSERT INTO __gtfs2sqlite_other_files (name, contents) VALUES (?, ?)", sqlitexNoop, filename, contents); err != nil {
			return err
		}
		return nil
	}

	inputCSV := csv.NewReader(inputF)
	table := strings.TrimSuffix(filename, ".txt")

	// Header

	header, err := inputCSV.Read()
	if err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Importing %s: %s", filename, strings.Join(header, ",")))

	var unknownColumns []string
	for _, column := range header {
		if _, ok := gtfsSchema[table].Columns[column]; !ok {
			unknownColumns = append(unknownColumns, column)
		}
	}
	_, hasTable := gtfsSchema[table]
	for _, column := range unknownColumns {
		columnFragment := column + " TEXT"

		var query string
		if hasTable {
			query = fmt.Sprintf("ALTER TABLE %s ADD %s", table, columnFragment)
		} else {
			query = fmt.Sprintf("CREATE TABLE %s (%s)", table, columnFragment)
			hasTable = true
		}

		if err := sqlitex.ExecTransient(db, query, sqlitexNoop); err != nil {
			return err
		}
	}

	var argFragments []string
	for i := range header {
		argFragments = append(argFragments, fmt.Sprintf("?%d", i+1))
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table, strings.Join(header, ", "), strings.Join(argFragments, ", "))
	insertStmt, err := db.Prepare(query)
	if err != nil {
		return err
	}

	// Rows

	inputCSV.FieldsPerRecord = -1 // Allow variable numbers of fields

	rowCount := 0
	for {
		row, err := inputCSV.Read()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}

		err = insertStmt.Reset()
		if err != nil {
			return err
		}
		err = insertStmt.ClearBindings()
		if err != nil {
			return err
		}

		for i, v := range row {
			param := i + 1
			if v == "" {
				insertStmt.BindNull(param)
			} else {
				insertStmt.BindText(param, v)
			}
		}

		for {
			rowReturned, err := insertStmt.Step()
			if err != nil {
				return err
			}
			if !rowReturned {
				break
			}
		}

		rowCount++
	}
	slog.Info(fmt.Sprintf("Wrote %d rows", rowCount))

	if rowCount == 0 {
		if err := sqlitex.Exec(db, "CREATE TABLE IF NOT EXISTS __gtfs2sqlite_empty_files (tableName TEXT)", sqlitexNoop); err != nil {
			return err
		}
		if err := sqlitex.Exec(db, "INSERT INTO __gtfs2sqlite_empty_files (tableName) VALUES (?)", sqlitexNoop, table); err != nil {
			return err
		}
	}

	return nil
}
