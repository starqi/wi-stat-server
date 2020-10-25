package main

import (
    "net/http"
    "log"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "github.com/gin-gonic/gin"
    hsql "github.com/starqi/wi-util-servers/cmd/stats/sql"
)

type HiscoreEntry struct {
    Name string `json:"name" binding:"required"`
    Team string `json:"team" binding:"required"`
    Kills int `json:"kills" binding:"required"`
    Deaths int `json:"deaths" binding:"required"`
}

var hdb *hsql.HiscoresDb

func main() {

    db, err := sql.Open("sqlite3", "./dist/db.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    hdb = hsql.MakeHiscoresDb(db)

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
    hiscores := hdb.Select()
    c.JSON(http.StatusOK, dbHiscoresToJson(hiscores))
}

// curl -d '[{"name":"hank","team":"sutasu","kills":0,"deaths":400}]' -H 'Content-Type:application/json' -u me:123 -i localhost:8080/hiscore
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
             h.Name, h.Team, h.Kills, h.Deaths,
        })
    }
    return a
}

func jsonHiscoresToDb(hiscores []HiscoreEntry) []hsql.HiscoreEntry {
    a := make([]hsql.HiscoreEntry, 0, len(hiscores))
    for _, h := range hiscores {
        a = append(a, hsql.HiscoreEntry {
             0, h.Name, h.Team, h.Kills, h.Deaths,
        })
    }
    return a
}
