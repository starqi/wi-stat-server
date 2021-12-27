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
    Kills int64 `json:"kills"`
    Deaths int64 `json:"deaths"`
    Bounty int64 `json:"bounty"`
    CharSpecName string `json:"charSpecName"`
    ExtraValues map[string]int64 `json:"extraValues"`
}

const cullTickerSeconds = 60
const topNToKeep = 10
const signatureHeader = "signature"
const relativeDbPathEnv = "relativeDbPath"
const postPublicKeyEnv = "postPublicKey"

var hdb *hsql.HiscoresDb
var cullTicker *time.Ticker
var postPublicKey *rsa.PublicKey

func cullTickerFunc() {
    /*
    for {
        <-cullTicker.C
        tx := hdb.MakeTransaction()
        rowsAffected := tx.Cull(topNToKeep)
        tx.Commit()
        if rowsAffected > 0 {
            log.Print("Culled ", rowsAffected, " rows")
        }
    }
    */
}

func main() {

    postPublicKeyHeader := os.Getenv(postPublicKeyEnv)
    if postPublicKeyHeader == "" {
        log.Printf("Missing %s, will not be able to update", postPublicKeyEnv)
    } else {
        _postPublicKeyIsolated, _ := pem.Decode([]byte(postPublicKeyHeader))
        _postPublicKeyParsed, err := x509.ParsePKCS1PublicKey(_postPublicKeyIsolated.Bytes)
        if err != nil {
            log.Print("Failed to parse POST public key, will not be able to update", err)
        }
        postPublicKey = _postPublicKeyParsed
        log.Print("Found POST public key")
    }

    relativeDbPath := os.Getenv(relativeDbPathEnv)
    if relativeDbPath == "" {
        log.Fatalf("Missing %s", relativeDbPath)
    }

    _hdb, err := hsql.MakeHiscoresDb(relativeDbPath)
    hdb = _hdb
    if err != nil {
        log.Fatal("Could not access DB", err)
    }

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
    router.Run() // Will use PORT env var
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

func getTopHiscores(c *gin.Context) {
    result, err := hdb.Transaction(func (tx *hsql.HiscoresDbTransaction) (interface{}, error) {
        return tx.Select(10, "kills")
    })
    if err != nil {
        log.Print("Failed to get top hiscores - ", err)
        c.Status(http.StatusInternalServerError)
        return
    }

    hiscores, ok := result.([]hsql.HiscoreWithMap)
    if !ok {
        log.Print("Unexpected cast error")
        c.Status(http.StatusInternalServerError)
        return
    }

    c.JSON(http.StatusOK, dbHiscoresToJson(hiscores))
}

func postHiscore(c *gin.Context) {
    var json []HiscoreEntry
    if err := c.ShouldBindJSON(&json); err != nil {
        c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    rowsAffected, err := hdb.Transaction(func (tx *hsql.HiscoresDbTransaction) (interface{}, error) {
        return tx.Insert(jsonHiscoresToDb(json))
    })
    if err != nil {
        log.Print("Failed to POST - ", err)
        c.Status(http.StatusInternalServerError)
        return
    }
    log.Printf("Posted %d rows", rowsAffected)
    c.Status(http.StatusOK)
}

func jsonHiscoresToDb(json []HiscoreEntry) []hsql.Hiscore {
    result := make([]hsql.Hiscore, 0, len(json))
    for _, j := range json {
        hiscoreValues := make([]hsql.HiscoreValue, 0)
        for key, val := range j.ExtraValues {
            hiscoreValues = append(hiscoreValues, hsql.HiscoreValue { Key: key, Value: val })
        }
        hiscoreValues = append(hiscoreValues, hsql.HiscoreValue { Key: "kills", Value: j.Kills })
        hiscoreValues = append(hiscoreValues, hsql.HiscoreValue { Key: "deaths", Value: j.Deaths })
        hiscoreValues = append(hiscoreValues, hsql.HiscoreValue { Key: "bounty", Value: j.Bounty })

        hiscoreData := make([]hsql.HiscoreData, 0)
        hiscoreData = append(hiscoreData, hsql.HiscoreData { Key: "team", Value: j.Team })
        hiscoreData = append(hiscoreData, hsql.HiscoreData { Key: "charSpecName", Value: j.CharSpecName })

        result = append(result, hsql.Hiscore {
            Name: j.Name,
            HiscoreValues: hiscoreValues,
            HiscoreData: hiscoreData,
        })
    }
    return result
}

func dbHiscoresToJson(hiscores []hsql.HiscoreWithMap) []HiscoreEntry {
    result := make([]HiscoreEntry, 0, len(hiscores))
    for _, h := range hiscores {
        
        extraValues := make(map[string]int64)
        for key, value := range h.ValueMap {
            extraValues[key] = value
        }
        delete(extraValues, "kills")
        delete(extraValues, "deaths")
        delete(extraValues, "bounty")

        result = append(result, HiscoreEntry {
            Name: h.Hiscore.Name,
            Team: h.DataMap["team"],
            Kills: h.ValueMap["kills"],
            Deaths: h.ValueMap["deaths"],
            Bounty: h.ValueMap["bounty"],
            CharSpecName: h.DataMap["charSpecName"],
            ExtraValues: extraValues,
        })
    }
    return result
}
