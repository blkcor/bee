package main

import (
	"fmt"
	beeorm "github.com/blkcor/beeORM"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	engine, _ := beeorm.NewEngine("sqlite3", "bee.db")
	defer engine.Close()
	s := engine.NewSession()
	_, _ = s.Raw("DROP TABLE IF EXISTS User;").Exec()
	_, _ = s.Raw("CREATE TABLE User(Name text);").Exec()
	result, _ := s.Raw("INSERT INTO User(`Name`) values (?), (?)", "Tom", "Sam").Exec()
	count, _ := result.RowsAffected()
	fmt.Printf("Exec success, %d affected\n", count)
	row := s.Raw("SELECT Name FROM User LIMIT 1").QueryRow()
	var name string
	_ = row.Scan(&name)
	fmt.Println("QueryRow -->", name)
	rows, _ := s.Raw("SELECT * FROM User").QueryRows()
	for rows.Next() {
		var name string
		_ = rows.Scan(&name)
		fmt.Println("QueryRows -->", name)
	}

}
