package sql

import (
    "fmt"
    "database/sql"
    "log"
    "time"
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
    Bounty int
    Timestamp *time.Time
}

func (r *HiscoreEntry) ToString() string {
    return fmt.Sprint(r.Id, " ", r.Name, " ", r.Team, " ", r.Kills, " ", r.Deaths, " ", r.Bounty, " ", r.Timestamp.String())
}

// FIXME Fatals

func (hdb *HiscoresDb) Select() []HiscoreEntry {
    rows, err := hdb.db.Query("select * from hiscores order by kills desc")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    results := make([]HiscoreEntry, 0)
    var unixTimeHolder int64;
    for rows.Next() {

        h := HiscoreEntry {}
        err = rows.Scan(&h.Id, &h.Name, &h.Team, &h.Kills, &h.Deaths, &h.Bounty, &unixTimeHolder)
        if err != nil {
            log.Fatal(err)
        }
        unixTime := time.Unix(unixTimeHolder, 0)
        h.Timestamp = &unixTime

        results = append(results, h)
    }
    err = rows.Err()
    if err != nil {
        log.Fatal(err)
    }
    return results
}

// PK is autoincrement, timestamp is generated and replaced
func (hdb *HiscoresDb) Insert(entries []HiscoreEntry) {
    stmt, err := hdb.db.Prepare("insert into hiscores (name, team, kills, deaths, bounty, timestamp) values (?, ?, ?, ?, ?, ?)")
    if err != nil {
        log.Fatal(err)
    }
    defer stmt.Close()

    for _, entry := range entries {
        _, err = stmt.Exec(entry.Name, entry.Team, entry.Kills, entry.Deaths, entry.Bounty, time.Now().Unix())
        if err != nil {
            log.Fatal(err)
        }
    }
}
