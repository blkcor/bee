package beeorm

import (
	"database/sql"
	"fmt"
	"github.com/blkcor/beeORM/dialect"
	"github.com/blkcor/beeORM/log"
	"github.com/blkcor/beeORM/session"
	"strings"
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

// different return a-b => 返回在a中但是不在b中的元素
func different(a []string, b []string) (diff []string) {
	mapB := make(map[string]bool, len(b))
	for _, v := range b {
		mapB[v] = true
	}
	for _, v := range a {
		if _, ok := mapB[v]; !ok {
			diff = append(diff, v)
		}
	}
	return
}

// Migrate the database
func (e *Engine) Migrate(value interface{}) error {
	_, err := e.Transaction(func(s *session.Session) (result interface{}, err error) {
		if !s.Model(value).HasTable() {
			log.Infof("table %s doesn't exist", s.RefTable().Name)
			return nil, s.CreateTable()
		}
		table := s.RefTable()
		// query the table columns
		rows, _ := s.Raw(fmt.Sprintf("SELECT * FROM %s LIMIT 1", table.Name)).QueryRows()
		columns, _ := rows.Columns()
		addCols := different(table.FieldNames, columns)
		delCols := different(columns, table.FieldNames)
		log.Infof("added cols %v", addCols)
		log.Infof("deleted cols %v", delCols)

		// add col
		for _, col := range addCols {
			field := table.GetField(col)
			sqlStr := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table.Name, field.Name, field.Type)
			if _, err = s.Raw(sqlStr).Exec(); err != nil {
				return
			}
		}

		//del col
		if len(delCols) == 0 {
			return
		}
		tmpName := "tmp_" + table.Name
		fieldStr := strings.Join(table.FieldNames, ", ")
		//1、create a new table from the old table
		//2、copy the data from the old table to the new table
		s.Raw(fmt.Sprintf("CREATE TABLE %s AS SELECT %s FROM %s;", tmpName, fieldStr, table.Name))
		//3、drop the old table
		s.Raw(fmt.Sprintf("DROP TABLE %s;", table.Name))
		//4、rename the new table to the old table
		s.Raw(fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", tmpName, table.Name))

		//execute the statement
		_, err = s.Exec()
		return
	})
	return err
}
