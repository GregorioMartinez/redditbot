package main

import (
	"database/sql"
	"time"
	"log"
)

func NewDatabase() *sql.DB{
	// Open DB connection
	db, err := sql.Open("sqlite3", "./wikibot.db")
	if err != nil {
		panic(err)
	}

	// Make sure table exists
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS \"blacklist\" (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, type TEXT, datetime INTEGER, reddit_id TEXT, comment_id TEXT)")
	if err != nil {
		log.Fatal(err)
	}

	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
	}

	// Check if DB has bot already in it
	q, err := db.Query("SELECT COUNT(*) FROM blacklist WHERE reddit_id = ?", "xurih")
	if err != nil {
		log.Fatal(err)
	}
	defer q.Close()

	var count int
	for q.Next() {
		err = q.Scan(&count)
	}

	// If bot user isn't there then seed the db
	if count == 0 {
		// populate
		seedStatement, err := db.Prepare("INSERT INTO blacklist (name, type, datetime, reddit_id) VALUES(?, ?, ?, ?)")
		if err != nil {
			log.Fatal(err)
		}

		//@TODO This should actually make a request to me.json to pull this info dynamically
		now := time.Now()
		_, err = seedStatement.Exec("WikipediaPoster", "user", now.Unix(), "xurih")
		if err != nil {
			panic(err)
		}

		log.Println("Seeded database with bot user")

	}

	return db
}