package sql

import (
    "fmt"
    "database/sql"
    "log"
    _ "github.com/mattn/go-sqlite3"
)

type HiscoresDb struct {
    db *sql.DB
}

func MakeHiscoresDb(db *sql.DB) *HiscoresDb {
    return &HiscoresDb { db }
}

type HiscoreEntry struct {
    Id int64
    Name string
    Team string
    Kills int
    Deaths int
}

func (r *HiscoreEntry) ToString() string {
    return fmt.Sprint(r.Id, " ", r.Name, " ", r.Team, " ", r.Kills, " ", r.Deaths)
}

// FIXME Fatals

func (hdb *HiscoresDb) Select() []HiscoreEntry {
    rows, err := hdb.db.Query("select * from hiscores")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    results := make([]HiscoreEntry, 0)
    for rows.Next() {
        h := HiscoreEntry {}
        err = rows.Scan(&h.Id, &h.Name, &h.Team, &h.Kills, &h.Deaths)
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

func (hdb *HiscoresDb) Insert(entries []HiscoreEntry) {
    stmt, err := hdb.db.Prepare("insert into hiscores (name, team, kills, deaths) values (?, ?, ?, ?)")
    if err != nil {
        log.Fatal(err)
    }
    defer stmt.Close()

    for _, entry := range entries {
        _, err = stmt.Exec(entry.Name, entry.Team, entry.Kills, entry.Deaths)
        if err != nil {
            log.Fatal(err)
        }
    }
}
