package session

import (
	"database/sql"
	"github.com/blkcor/beeORM/clause"
	"github.com/blkcor/beeORM/dialect"
	"github.com/blkcor/beeORM/log"
	"github.com/blkcor/beeORM/schema"
	"strings"
)

// Session is responsible for interacting with the database
type Session struct {
	db       *sql.DB
	tx       *sql.Tx
	dialect  dialect.Dialect
	refTable *schema.Schema
	sql      strings.Builder
	sqlVars  []interface{}
	clause   clause.Clause
}

// CommonDB is the minimal functional set of DB
type CommonDB interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

var _ CommonDB = (*sql.DB)(nil)
var _ CommonDB = (*sql.Tx)(nil)

// New return an instance of Session struct
func New(db *sql.DB, dialect dialect.Dialect) *Session {
	return &Session{
		db:      db,
		dialect: dialect,
	}
}

// Clear reset the sql string and sqlVars
func (s *Session) clear() {
	s.sql.Reset()
	s.sqlVars = nil
	s.clause = clause.Clause{}
}

// DB return the sql.DB instance if tx is nil, otherwise return the sql.Tx instance
func (s *Session) DB() CommonDB {
	if s.tx != nil {
		return s.tx
	}
	return s.db
}

func (s *Session) Raw(sql string, values ...interface{}) *Session {
	s.sql.WriteString(sql)
	s.sql.WriteString(" ")
	s.sqlVars = append(s.sqlVars, values...)
	return s
}

// Exec execute the sql statement
func (s *Session) Exec() (result sql.Result, err error) {
	defer s.clear()
	log.Info(s.sql.String(), s.sqlVars)
	// sql lib support dynamic arguments
	// eg: result, err := db.Exec("INSERT INTO User(`Name`) values (?), (?)", "Tom", "Sam")
	if result, err = s.DB().Exec(s.sql.String(), s.sqlVars...); err != nil {
		log.Error(err)
	}
	return
}

// QueryRow query single row data
func (s *Session) QueryRow() *sql.Row {
	defer s.clear()
	log.Info(s.sql.String(), s.sqlVars)
	return s.DB().QueryRow(s.sql.String(), s.sqlVars...)
}

// QueryRows query multiple rows data
func (s *Session) QueryRows() (rows *sql.Rows, err error) {
	defer s.clear()
	log.Info(s.sql.String(), s.sqlVars)
	if rows, err = s.DB().Query(s.sql.String(), s.sqlVars...); err != nil {
		log.Error(err)
	}
	return
}
