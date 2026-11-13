package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type explorerColumn struct {
	name, kind              string
	nullable, primary, auto bool
	defaultValue            any
}
type explorerTable struct {
	name    string
	columns []explorerColumn
	primary string
}
type dbExplorer struct {
	db     *sql.DB
	tables map[string]explorerTable
	names  []string
}

// NewDBExplorer reads the schema once; request handlers share immutable metadata.
func NewDBExplorer(db *sql.DB) (http.Handler, error) {
	e := &dbExplorer{db: db, tables: make(map[string]explorerTable), names: make([]string, 0)}
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		e.names = append(e.names, name)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, name := range e.names {
		table, err := e.readTable(name)
		if err != nil {
			return nil, err
		}
		e.tables[name] = table
	}
	return e, nil
}
func quoteIdentifier(name string) string { return "`" + strings.ReplaceAll(name, "`", "``") + "`" }
func (e *dbExplorer) readTable(name string) (explorerTable, error) {
	t := explorerTable{name: name}
	rows, err := e.db.Query("SHOW FULL COLUMNS FROM " + quoteIdentifier(name))
	if err != nil {
		return t, err
	}
	defer rows.Close()
	names, err := rows.Columns()
	if err != nil {
		return t, err
	}
	for rows.Next() {
		values := make([]sql.NullString, len(names))
		dest := make([]any, len(names))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return t, err
		}
		fields := make(map[string]sql.NullString)
		for i, key := range names {
			fields[key] = values[i]
		}
		c := explorerColumn{name: fields["Field"].String, kind: columnKind(fields["Type"].String), nullable: fields["Null"].String == "YES", primary: fields["Key"].String == "PRI", auto: strings.Contains(fields["Extra"].String, "auto_increment")}
		if fields["Default"].Valid {
			c.defaultValue = fields["Default"].String
		}
		t.columns = append(t.columns, c)
		if c.primary {
			t.primary = c.name
		}
	}
	return t, rows.Err()
}
func columnKind(t string) string {
	t = strings.ToLower(t)
	if i := strings.IndexAny(t, "( "); i >= 0 {
		t = t[:i]
	}
	switch t {
	case "tinyint", "smallint", "mediumint", "int", "integer", "bigint", "year":
		return "int"
	case "float", "double", "decimal", "numeric", "real":
		return "float"
	default:
		return "string"
	}
}
func explorerJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func explorerError(w http.ResponseWriter, status int, message string) {
	explorerJSON(w, status, map[string]any{"error": message})
}
func explorerResponse(w http.ResponseWriter, value any) {
	explorerJSON(w, http.StatusOK, map[string]any{"response": value})
}
func (e *dbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	if path == "" && r.Method == http.MethodGet {
		explorerResponse(w, map[string]any{"tables": e.names})
		return
	}
	parts := strings.Split(path, "/")
	t, ok := e.tables[parts[0]]
	if !ok {
		explorerError(w, 404, "unknown table")
		return
	}
	if len(parts) > 2 {
		explorerError(w, 404, "record not found")
		return
	}
	var id int64
	if len(parts) == 2 {
		var err error
		id, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || t.primary == "" {
			explorerError(w, 404, "record not found")
			return
		}
	}
	var err error
	switch {
	case r.Method == http.MethodGet:
		err = e.read(w, r, t, len(parts) == 2, id)
	case r.Method == http.MethodPut && len(parts) == 1:
		err = e.write(w, r, t, false, id)
	case r.Method == http.MethodPost && len(parts) == 2:
		err = e.write(w, r, t, true, id)
	case r.Method == http.MethodDelete && len(parts) == 2:
		var result sql.Result
		result, err = e.db.ExecContext(r.Context(), "DELETE FROM "+quoteIdentifier(t.name)+" WHERE "+quoteIdentifier(t.primary)+" = ?", id)
		if err == nil {
			var n int64
			n, err = result.RowsAffected()
			if err == nil {
				explorerResponse(w, map[string]any{"deleted": n})
			}
		}
	default:
		explorerError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
	if err != nil {
		explorerError(w, 500, "internal server error")
	}
}
func pagination(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}
func (e *dbExplorer) read(w http.ResponseWriter, r *http.Request, t explorerTable, single bool, id int64) error {
	query := "SELECT * FROM " + quoteIdentifier(t.name)
	var args []any
	if single {
		query += " WHERE " + quoteIdentifier(t.primary) + " = ?"
		args = []any{id}
	} else {
		query += " LIMIT ? OFFSET ?"
		args = []any{pagination(r.URL.Query().Get("limit"), 5), pagination(r.URL.Query().Get("offset"), 0)}
	}
	rows, err := e.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	names, err := rows.Columns()
	if err != nil {
		return err
	}
	kinds := make(map[string]string)
	for _, c := range t.columns {
		kinds[c.name] = c.kind
	}
	records := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(names))
		dest := make([]any, len(names))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return err
		}
		record := make(map[string]any)
		for i, name := range names {
			value := values[i]
			if b, ok := value.([]byte); ok {
				value = string(b)
			}
			if s, ok := value.(string); ok {
				switch kinds[name] {
				case "int":
					if strings.HasPrefix(s, "-") {
						value, err = strconv.ParseInt(s, 10, 64)
					} else {
						value, err = strconv.ParseUint(s, 10, 64)
					}
				case "float":
					value, err = strconv.ParseFloat(s, 64)
				}
				if err != nil {
					return err
				}
			}
			record[name] = value
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if single {
		if len(records) == 0 {
			explorerError(w, 404, "record not found")
		} else {
			explorerResponse(w, map[string]any{"record": records[0]})
		}
	} else {
		explorerResponse(w, map[string]any{"records": records})
	}
	return nil
}
func validateExplorerValue(c explorerColumn, value any) (any, bool) {
	if value == nil {
		return nil, c.nullable
	}
	switch c.kind {
	case "int":
		n, ok := value.(json.Number)
		if !ok {
			return nil, false
		}
		if strings.HasPrefix(string(n), "-") {
			v, err := strconv.ParseInt(string(n), 10, 64)
			return v, err == nil
		}
		v, err := strconv.ParseUint(string(n), 10, 64)
		return v, err == nil
	case "float":
		n, ok := value.(json.Number)
		if !ok {
			return nil, false
		}
		v, err := n.Float64()
		return v, err == nil
	default:
		v, ok := value.(string)
		return v, ok
	}
}
func (e *dbExplorer) write(w http.ResponseWriter, r *http.Request, t explorerTable, update bool, id int64) error {
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	var body map[string]any
	if err := decoder.Decode(&body); err != nil || body == nil {
		explorerError(w, 400, "invalid request body")
		return nil
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		explorerError(w, 400, "invalid request body")
		return nil
	}
	var columns, placeholders []string
	var args []any
	var primaryValue any
	var autoPrimary bool
	for _, c := range t.columns {
		value, exists := body[c.name]
		if update && !exists {
			continue
		}
		if !update && c.auto {
			if c.primary {
				autoPrimary = true
			}
			continue
		}
		if update && c.primary {
			explorerError(w, 400, fmt.Sprintf("field %s have invalid type", c.name))
			return nil
		}
		if !exists {
			if c.defaultValue != nil {
				if c.primary {
					primaryValue = c.defaultValue
					if c.kind != "string" {
						primaryValue = json.Number(fmt.Sprint(c.defaultValue))
					}
				}
				continue
			}
			switch {
			case c.nullable:
				value = nil
			case c.kind == "string":
				value = ""
			default:
				value = json.Number("0")
			}
		}
		value, valid := validateExplorerValue(c, value)
		if !valid {
			explorerError(w, 400, fmt.Sprintf("field %s have invalid type", c.name))
			return nil
		}
		if c.primary {
			primaryValue = value
		}
		name := quoteIdentifier(c.name)
		if update {
			name += " = ?"
		}
		columns = append(columns, name)
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}
	if update && len(columns) == 0 {
		explorerResponse(w, map[string]any{"updated": 0})
		return nil
	}
	query := "INSERT INTO " + quoteIdentifier(t.name) + " (" + strings.Join(columns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"
	if update {
		query = "UPDATE " + quoteIdentifier(t.name) + " SET " + strings.Join(columns, ", ") + " WHERE " + quoteIdentifier(t.primary) + " = ?"
		args = append(args, id)
	}
	result, err := e.db.ExecContext(r.Context(), query, args...)
	if err != nil {
		return err
	}
	if update {
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		explorerResponse(w, map[string]any{"updated": n})
	} else {
		if autoPrimary {
			n, err := result.LastInsertId()
			if err != nil {
				return err
			}
			primaryValue = n
		}
		explorerResponse(w, map[string]any{t.primary: primaryValue})
	}
	return nil
}
