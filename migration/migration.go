package main

import (
    "database/sql"
    "log"
    _ "github.com/mattn/go-sqlite3"
)

func main() {
    createDb()
}

func createDb() *sql.DB {
    db, err := sql.Open("sqlite3", "./db/db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    _, err = db.Exec(`
        create table hiscores (
            id integer not null primary key autoincrement,
            name text,
            team text,
            kills integer,
            deaths integer
        )
    `)
    if err != nil {
        log.Fatal(err)
    }
    return db
}
