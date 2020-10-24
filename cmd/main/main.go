package main

import (
    "fmt"
    "database/sql"
    "log"
    _ "github.com/mattn/go-sqlite3"
    "github.com/gin-gonic/gin"
)

type HiscoreEntry struct {
    Id int64
    Name string
    Team string
    Kills int
    Deaths int
}

func (r *HiscoreEntry) toString() string {
    return fmt.Sprint(r.Id, " ", r.Name, " ", r.Team, " ", r.Kills, " ", r.Deaths)
}

func main() {

    db, err := sql.Open("sqlite3", "./db/db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    router := gin.Default()

    router.GET("/hiscores", func(c *gin.Context) {
        hiscores := selectAll(db)
        c.JSON(200, dbHiscoresToJson(hiscores))
    })

    router.Run()
}

func dbHiscoresToJson(hiscores []HiscoreEntry) []gin.H {
    a := make([]gin.H, 0, len(hiscores))
    for _, h := range hiscores {
        a = append(a, gin.H {
            "name": h.Name,
            "team": h.Team,
            "kills": h.Kills,
            "deaths": h.Deaths,
        })
    }
    return a
}

func selectAll(db *sql.DB) []HiscoreEntry {
    rows, err := db.Query("select * from hiscores")
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

func insertMany(db *sql.DB, entries []HiscoreEntry) {
    stmt, err := db.Prepare("insert into hiscores (name, team, kills, deaths) values (?, ?, ?, ?)")
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
