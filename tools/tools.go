package tools

import (
	"database/sql"
	"fmt"

	"github.com/gidoBOSSftw5731/log"
)

//StartSQL returns a database
func StartSQL(user, password, ip, port string) *sql.DB {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/pinnerboibot",
		user, password, ip, port))
	if err != nil {
		log.Error("Oh noez, could not connect to database")
		log.Errorf("Error in SQL! %v", err)
	}
	log.Debug("Oi, mysql did thing")
	//defer db.Close()

	return db
}
