// The MIT License (MIT)
//
// Copyright (c) 2015 Arnaud Vazard
//
// See LICENSE file.
package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

func InitDatabase(databaseName string, reset bool) *sql.DB {
	// Use default name if not specified
	if databaseName == "" {
		// check if the storage directory exist, if not create it
		storage, err := os.Stat("./storage")
		if err != nil {
			os.Mkdir("./storage", os.ModeDir)
		} else if !storage.IsDir() {
			// check if the storage is indeed a directory or not
			log.Fatal("\"storage\" exist but is not a directory")
		}
		databaseName = "./storage/db.sqlite"
	}

	if reset {
		os.Remove(databaseName)
	}

	db, err := sql.Open("sqlite3", databaseName)
	if err != nil {
		log.Fatal(err)
	}
	return db
}