package beeorm

import (
	"database/sql"
	"github.com/blkcor/beeORM/dialect"
	"github.com/blkcor/beeORM/log"
	"github.com/blkcor/beeORM/session"
)

type TxFunc func(*session.Session) (interface{}, error)

// Engine provider user interaction with the database
type Engine struct {
	db      *sql.DB
	dialect dialect.Dialect
}

// NewEngine creates a new database connection
func NewEngine(driver, dataSource string) (e *Engine, err error) {
	db, err := sql.Open(driver, dataSource)
	if err != nil {
		log.Error(err)
		return
	}

	// Send a ping to make sure the db connection is alive
	if err = db.Ping(); err != nil {
		log.Error(err)
		return
	}
	dia, ok := dialect.GetDialect(driver)
	if !ok {
		log.Errorf("dialect %s Not Found", driver)
		return
	}
	e = &Engine{db: db, dialect: dia}
	log.Info("Connect to the database successful!")
	return
}

// Transaction starts a transaction
func (e *Engine) Transaction(fn TxFunc) (result interface{}, err error) {
	s := e.NewSession()
	if err = s.Begin(); err != nil {
		return
	}
	defer func() {
		// transaction 发生panic时，进行回滚
		if p := recover(); p != nil {
			_ = s.RollBack()
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			_ = s.RollBack() // err is non-nil; don't change it
		} else {
			// no exception ==> just commit the transaction
			err = s.Commit()
		}
	}()
	return fn(s)
}

// Close the database connection
func (e *Engine) Close() {
	if err := e.db.Close(); err != nil {
		log.Error("Failed to close database")
	}
	log.Info("Close database successful!")
}

// NewSession creates a new session for database operations
func (e *Engine) NewSession() *session.Session {
	return session.New(e.db, e.dialect)
}
