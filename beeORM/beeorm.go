package beeorm

import (
	"database/sql"
	"github.com/blkcor/beeORM/log"
	"github.com/blkcor/beeORM/session"
)

// Engine provider user interaction with the database
type Engine struct {
	db *sql.DB
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
	e = &Engine{db: db}
	log.Info("Connect to the database successful!")
	return
}

// Close the database connection
func (e *Engine) Close() {
	if err := e.db.Close(); err != nil {
		log.Error("Failed to close database!")
	}
	log.Info("Close database successful!")
}

// NewSession creates a new session for database operations
func (e *Engine) NewSession() *session.Session {
	return session.New(e.db)
}
