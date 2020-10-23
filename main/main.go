package main

import (
    "database/sql"
    "log"
    _ "github.com/mattn/go-sqlite3"
)

type hiscoreEntry struct {
    id int64
    name string
    team string
    kills int
    deaths int
}

func main() {
    db, err := sql.Open("sqlite3", "./db/db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    var entries = []hiscoreEntry {
        hiscoreEntry { 0, "Hank", "sutasu ind", 4, 20 },
        hiscoreEntry { 0, "AxCWs", "sutasu ind", 0, 69 },
        hiscoreEntry { 0, "Moebius", "sutasu ind", 13, 37 },
        hiscoreEntry { 0, "Bank", "sutasu ind", 12, 34 },
    }
    insertMany(db, entries)
    hiscores := selectAll(db)
    for _, h := range hiscores {
        log.Print(h.id, " ", h.name, " ", h.team, " ", h.kills, " ", h.deaths)
    }
}

func selectAll(db *sql.DB) []hiscoreEntry {
    rows, err := db.Query("select * from hiscores")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    results := make([]hiscoreEntry, 0)
    for rows.Next() {
        h := hiscoreEntry {}
        err = rows.Scan(&h.id, &h.name, &h.team, &h.kills, &h.deaths)
        if err != nil {
            log.Fatal(err)
        }
        results = append(results, h)
    }
    err = rows.Err()
    if err != nil {
        log.Fatal(err)
    }
    return results
}

func insertMany(db *sql.DB, entries []hiscoreEntry) {
    stmt, err := db.Prepare("insert into hiscores (name, team, kills, deaths) values (?, ?, ?, ?)")
    if err != nil {
        log.Fatal(err)
    }
    defer stmt.Close()

    for _, entry := range entries {
        _, err = stmt.Exec(entry.name, entry.team, entry.kills, entry.deaths)
        if err != nil {
            log.Fatal(err)
        }
    }
}
