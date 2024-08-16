package gtfs2sqlite

import (
	"context"
	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

var ErrInvalidInput = errors.New("invalid input")

type validateOpts struct {
	force    bool
	ignore   bool
	logLevel slog.Level
}

func validate(db *sqlite.Conn, opts validateOpts) ([]string, error) {
	v := &validator{db: db, opts: opts, toDelete: make(map[string][]int64)}

	slog.Info("Validating")

	for {
		for table, schema := range gtfsSchema {
			if err := v.validateTable(table, schema); err != nil {
				return nil, err
			}
		}
		if len(v.toDelete) == 0 {
			break
		}

		deleted := 0
		for table, rows := range v.toDelete {
			query := fmt.Sprintf("DELETE FROM %s WHERE rowid = ?", table)
			for _, rowid := range rows {
				if err := sqlitex.Exec(db, query, sqlitexNoop, rowid); err != nil {
					return nil, err
				}
				deleted++
			}
		}
		slog.Info(fmt.Sprintf("Re-validating after force deleting %d row(s)", deleted))
		v.toDelete = make(map[string][]int64)
		v.pass++
	}

	if len(v.issues) > 0 {
		if opts.force || opts.ignore {
			return v.issues, nil
		} else {
			return v.issues, ErrInvalidInput
		}
	}
	return nil, nil
}

type validator struct {
	db       *sqlite.Conn
	opts     validateOpts
	issues   []string
	pass     int
	toDelete map[string][]int64 // table -> rowid
}

func (v *validator) append(msg string, args ...any) {
	issue := fmt.Sprintf(msg, args...)
	slog.Log(context.Background(), v.opts.logLevel, issue)
	v.issues = append(v.issues, issue)
}

func (v *validator) validateTable(table string, schema tableSchema) error {
	for column, schema := range schema.Columns {
		if err := v.validateColumn(table, column, schema); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateColumn(table, column string, schema columnSchema) error {
	if schema.ForeignID != nil {
		if err := v.validateForeignID(table, column, *schema.ForeignID); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateForeignID(table, column string, schema foreignIDSchema) error {
	// Normalize to AnyOf form
	if len(schema.AnyOf) > 0 {
		if schema.Table != "" || schema.Column != "" {
			panic("If AnyOf cannot have Table or Column")
		}
	} else {
		schema.AnyOf = []foreignIDSchema{{Table: schema.Table, Column: schema.Column}}
		schema.Table = ""
		schema.Column = ""
	}

	var foreignFragments []string
	for _, subSchema := range schema.AnyOf {
		fragment := fmt.Sprintf("SELECT %s FROM %s", subSchema.Column, subSchema.Table)
		foreignFragments = append(foreignFragments, fragment)
	}
	foreignFragment := strings.Join(foreignFragments, " UNION ")

	query := fmt.Sprintf("SELECT rowid, * FROM %s WHERE %s IS NOT NULL AND %s NOT IN (%s)",
		table, column, column, foreignFragment)

	return sqlitex.Exec(v.db, query, func(stmt *sqlite.Stmt) error {
		rowid := stmt.GetInt64("rowid")
		value := stmt.GetText(column)

		if v.pass == 0 {
			v.append("%s in %s.txt is not a valid %s [%s]", value, table, column, prettyPrintRow(stmt))
		}

		if v.opts.force {
			v.toDelete[table] = append(v.toDelete[table], rowid)
		}

		return nil
	})
}

func prettyPrintRow(row *sqlite.Stmt) string {
	var out []string
	for i := range row.ColumnCount() {
		column := row.ColumnName(i)
		value := row.GetText(column)
		if column != "rowid" && value != "" {
			out = append(out, fmt.Sprintf("%s: %s", column, value))
		}
	}
	return strings.Join(out, ", ")
}
