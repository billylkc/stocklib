package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func GetConnection() (*sql.DB, error) {
	secret := os.Getenv("STOCK_CONNECT")
	if secret == "" {
		log.Fatal(fmt.Errorf("missing environment variable STOCK_CONNECT. Please check."))
	}

	db, err := sql.Open("postgres", secret)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func RecordExists(table, date string) bool {
	db, err := GetConnection()
	if err != nil {
		return true
	}
	queryF := `
    SELECT count(1) as cnt
    FROM %s
    WHERE date = '%s'`

	query := fmt.Sprintf(queryF, table, date)
	rows, err := db.Query(query)
	defer rows.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	var num int
	for rows.Next() {
		_ = rows.Scan(&num)
	}
	if num > 0 {
		return true
	}
	return false // false as safe to insert
}
