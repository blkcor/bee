package session

import "github.com/blkcor/beeORM/log"

// Begin a transaction
func (s *Session) Begin() (err error) {
	log.Info("Begin Transaction......")
	if s.tx, err = s.db.Begin(); err != nil {
		log.Error(err)
	}
	return
}

// Commit a transaction
func (s *Session) Commit() (err error) {
	log.Info("Commit Transaction......")
	if err = s.tx.Commit(); err != nil {
		log.Error(err)
	}
	return
}

// RollBack a transaction
func (s *Session) RollBack() (err error) {
	log.Info("RollBack Transaction......")
	if err := s.tx.Rollback(); err != nil {
		log.Error(err)
	}
	return
}
