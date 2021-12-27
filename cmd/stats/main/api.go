package main

import (
    "time"
    "net/http"
    "log"
    "github.com/gin-gonic/gin"
    "os"
    "crypto/rsa"
    "crypto/x509"
    "crypto/sha256"
    "encoding/pem"
    "crypto"
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
    CharSpecName string `json:"charSpecName"`
    ExtraValues map[string]string `json:"extraValues"`
}

const cullTickerSeconds = 60
const topNToKeep = 10
const signatureHeader = "signature"

var hdb *hsql.HiscoresDb
var cullTicker *time.Ticker
var postPublicKey *rsa.PublicKey

func cullTickerFunc() {
    for {
        <-cullTicker.C
        /*
        rowsAffected := hdb.Cull(topNToKeep)
        if rowsAffected > 0 {
            log.Print("Culled ", rowsAffected, " rows")
        }
        */
    }
}

func main() {

    postPublicKeyHeader := os.Getenv("POST_PUBLIC_KEY")
    if postPublicKeyHeader == "" {
        log.Print("No POST public key found, will not be able to update")
    } else {
        _postPublicKeyIsolated, _ := pem.Decode([]byte(postPublicKeyHeader))
        _postPublicKeyParsed, err := x509.ParsePKCS1PublicKey(_postPublicKeyIsolated.Bytes)
        if err != nil {
            log.Print("Failed to parse POST public key, will not be able to update", err)
        }
        postPublicKey = _postPublicKeyParsed
        log.Print("Found POST public key")
    }

    hdb, err := hsql.MakeHiscoresDb("./dist/db.db")
    if err != nil {
        log.Fatal("Could not access DB", err)
    }

    hdb.Insert([]hsql.Hiscore {
        {
            Name: "Bob",
            HiscoreValues: []hsql.HiscoreValue {
                { Key: "Kills", Value: 1 },
                { Key: "Deaths", Value: 2 },
                { Key: "IQ", Value: 3 },
            },
        },
    })

    results, err := hdb.Select(2, "Kills")
    if err != nil {
        log.Fatal(err)
    }

    log.Print(results)

    /*
    cullTicker = time.NewTicker(cullTickerSeconds * time.Second)
    go cullTickerFunc()

    router := gin.Default()
    router.Use(func (c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "*")
        c.Header("Access-Control-Allow-Headers", "Authorization, *")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(http.StatusNoContent)
        } else {
            c.Next()
        }
    })

    basicAuthGroup := router.Group("/hiscore", checkSignature)
    basicAuthGroup.POST("", postHiscore)

    router.GET("/top", getTopHiscores)
    router.Run()
    */
}

func checkSignature(c * gin.Context) {
    if postPublicKey == nil {
        c.Status(http.StatusUnauthorized)
    } else {
        rawData, err := c.GetRawData()
        if err != nil {
            log.Print("Failed to get raw data")
            c.Status(http.StatusInternalServerError)
            return
        }
        hash := sha256.Sum256(rawData)
        sig := c.GetHeader(signatureHeader)
        err = rsa.VerifyPKCS1v15(postPublicKey, crypto.SHA256, hash[:], []byte(sig))
        if err != nil {
            c.Status(http.StatusUnauthorized)
        }
    }
}

/*
func getTopHiscores(c *gin.Context) {
    hiscores, err := hdb.Select(10, "kills")
    if err != nil {
        log.Print("Failed to get top hiscores", err)
        c.Status(http.StatusInternalServerError)
        return
    }
    //c.JSON(http.StatusOK, dbHiscoresToJson(hiscores))
}

// FIXME
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
    result := make([]HiscoreEntry, 0, len(hiscores))
    for _, h := range hiscores {
        extraValues := make(map[string]string)
        for _, extraValue := range h.ExtraValues {
            extraValues[extraValue.Name] = extraValue.Value
        }
        result = append(result, HiscoreEntry {
            h.Name, h.Team, h.Kills, h.Deaths, h.Bounty, h.Timestamp.Format(time.RFC1123), "FIXME", extraValues,
        })
    }
    return result
}

func jsonHiscoresToDb(hiscores []HiscoreEntry) []hsql.HiscoreEntry {
    entries := make([]hsql.HiscoreEntry, 0, len(hiscores))
    for _, h := range hiscores {
        extraValues := make([]hsql.ExtraValue, len(h.ExtraValues))
        for key, val := range h.ExtraValues {
            extraValues = append(extraValues, hsql.ExtraValue { 0, key, val })
        }
        entries = append(entries, hsql.HiscoreEntry {
             0, h.Name, h.Team, h.Kills, h.Deaths, h.Bounty, nil, extraValues,
        })
    }
    return entries
}
*/
