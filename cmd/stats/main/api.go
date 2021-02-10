package main

import (
    "time"
    "net/http"
    "log"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "github.com/gin-gonic/gin"
    hsql "github.com/starqi/wi-util-servers/cmd/stats/sql"
)

// Required does not work unless value can contain nil?
type HiscoreEntry struct {
    Name string `json:"name"`
    Team string `json:"team"`
    Kills int `json:"kills"`
    Deaths int `json:"deaths"`
    Bounty int `json:"bounty"`
    Timestamp string `json:"timestamp"`
}

const cullTickerSeconds = 60
const topNToKeep = 10

// FIXME Pass thru params
var hdb *hsql.HiscoresDb
var cullTicker *time.Ticker

func cullTickerFunc() {
    for {
        <-cullTicker.C
        rowsAffected := hdb.Cull(topNToKeep)
        if rowsAffected > 0 {
            log.Print("Culled ", rowsAffected, " rows")
        }
    }
}

func main() {

    db, err := sql.Open("sqlite3", "./dist/db.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    hdb = hsql.MakeHiscoresDb(db)

    cullTicker = time.NewTicker(cullTickerSeconds * time.Second)
    go cullTickerFunc()

    router := gin.Default()
    router.Use(func (c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "http://localhost:8081")
        c.Header("Access-Control-Allow-Methods", "*")
        c.Header("Access-Control-Allow-Credentials", "true")
        c.Header("Access-Control-Allow-Headers", "Authorization, *")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(http.StatusNoContent)
        } else {
            c.Next()
        }
    })

    basicAuthGroup := router.Group("/hiscore", gin.BasicAuth(gin.Accounts {
        "me": "123",
    }))
    basicAuthGroup.GET("/top", getTopHiscores)
    basicAuthGroup.POST("", postHiscore)
    router.Run()
}

func getTopHiscores(c *gin.Context) {
    hiscores := hdb.Select(10)
    c.JSON(http.StatusOK, dbHiscoresToJson(hiscores))
}

// curl -d '[{"name":"hank","team":"sutasu","kills":0,"deaths":400,"bounty":38}]' -H 'Content-Type:application/json' -u me:123 -i localhost:8080/hiscore
func postHiscore(c *gin.Context) {
    var json []HiscoreEntry
    if err := c.ShouldBindJSON(&json); err != nil {
        c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    hdb.Insert(jsonHiscoresToDb(json))
    c.Status(http.StatusOK)
}

// curl -i -u me:123 localhost:8080/hiscore/top
func dbHiscoresToJson(hiscores []hsql.HiscoreEntry) []HiscoreEntry {
    a := make([]HiscoreEntry, 0, len(hiscores))
    for _, h := range hiscores {
        a = append(a, HiscoreEntry {
            h.Name, h.Team, h.Kills, h.Deaths, h.Bounty, h.Timestamp.Format(time.RFC1123),
        })
    }
    return a
}

func jsonHiscoresToDb(hiscores []HiscoreEntry) []hsql.HiscoreEntry {
    a := make([]hsql.HiscoreEntry, 0, len(hiscores))
    for _, h := range hiscores {
        a = append(a, hsql.HiscoreEntry {
             0, h.Name, h.Team, h.Kills, h.Deaths, h.Bounty, nil,
        })
    }
    return a
}
