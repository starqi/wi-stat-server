package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	hsql "github.com/starqi/wi-util-servers/cmd/stats/sql"
    decrypt "github.com/starqi/wi-util-servers/cmd/stats/decrypt"
)

// Required does not work unless value can contain nil?
type HiscoreEntry struct {
    Name string `json:"name"`
    Team string `json:"team"`
    Kills int64 `json:"kills"`
    Deaths int64 `json:"deaths"`
    ClassName string `json:"className"`
    Bounty int64 `json:"bounty"`
    BountyColor string `json:"bountyColor"`
    ExtraValues map[string]int64 `json:"extraValues"`
    ExtraData map[string]string `json:"extraData"`
}

const maxTopHiscores = 10
const cullTickerSeconds = 60
const topNToKeep = 10
const relativeDbPathEnv = "relativeDbPath"
const sharedSecretEnv = "sharedSecret"

var cullColumns = []string{ "kills", "healed", "bounty" }
var hdb *hsql.HiscoresDb
var cullTicker *time.Ticker
var sharedSecret []byte

func cullTickerFunc() {
    for {
        <-cullTicker.C
        _, err := hdb.Transaction(func (tx *hsql.HiscoresDbTransaction) (interface{}, error) {
            return tx.Cull(topNToKeep, cullColumns)
        })
        if err != nil {
            log.Print("Failed to cull, rolled back - ", err)
        }
    }
}

func main() {

    sharedSecretInput := os.Getenv(sharedSecretEnv)
    if sharedSecretInput == "" {
        log.Printf("Missing %s, will not be able to update hiscores", sharedSecretEnv)
    } else {
        _sharedSecret, err := base64.StdEncoding.DecodeString(sharedSecretInput)
        if err != nil {
            log.Print("Failed to parse shared secret, will not be able to update hiscores")
        } else {
            sharedSecret = _sharedSecret
            log.Print("Found shared secret")
        }
    }

    relativeDbPath := os.Getenv(relativeDbPathEnv)
    if relativeDbPath == "" {
        log.Fatalf("Missing %s", relativeDbPathEnv)
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
    router.POST("hiscore", postHiscore)
    router.GET("hiscore/top", getTopHiscores)
    router.Run() // Will use PORT env var
}

func getTopHiscores(c *gin.Context) {
    field := c.Query("field")
    if field == "" {
        c.Status(http.StatusBadRequest)
        c.Writer.Write([]byte("Missing field param"))
        return
    }

    _num := c.Query("num")
    num, err := strconv.Atoi(_num)
    if err != nil || num > maxTopHiscores {
        num = maxTopHiscores
    }

    _by := c.Query("by")
    by, err := strconv.Atoi(_by)
    if err != nil || by < 0 || by > 2 {
        by = 0
    }

    result, err := hdb.Transaction(func (tx *hsql.HiscoresDbTransaction) (interface{}, error) {
        return tx.Select(num, field, hsql.TimeGroupSeconds[by])
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
    rawData, err := ioutil.ReadAll(c.Request.Body)
    if err != nil {
        log.Print("Body parse error ", err)
        c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // Must manually ensure body reader is not stuck at EOF 
    c.Request.Body = ioutil.NopCloser(bytes.NewReader(rawData))

    binaryData := make([]byte, base64.StdEncoding.DecodedLen(len(string(rawData))))
    n, err := base64.StdEncoding.Decode(binaryData, rawData)
    binaryData = binaryData[:n]
    if err != nil {
        log.Print("Base64 error ", err)
        c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    payload, err := decrypt.DecryptHandlePostedHiscores(sharedSecret, binaryData)
    if err != nil {
        log.Print("Decrypt failed! ", err, " ", string(rawData))
        c.AbortWithStatus(http.StatusUnauthorized)
        return
    }
    payloadStr := string(payload)
    log.Print(payloadStr)

    var hiscores []HiscoreEntry
    err = json.Unmarshal(payload, &hiscores)
    if err != nil {
        c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    rowsAffected, err := hdb.Transaction(func (tx *hsql.HiscoresDbTransaction) (interface{}, error) {
        return tx.Insert(jsonHiscoresToDb(hiscores))
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
        for key, val := range j.ExtraData {
            hiscoreData = append(hiscoreData, hsql.HiscoreData { Key: key, Value: val })
        }
        hiscoreData = append(hiscoreData, hsql.HiscoreData { Key: "team", Value: j.Team })
        hiscoreData = append(hiscoreData, hsql.HiscoreData { Key: "className", Value: j.ClassName })
        hiscoreData = append(hiscoreData, hsql.HiscoreData { Key: "bountyColor", Value: j.BountyColor })

        result = append(result, hsql.Hiscore {
            Name: j.Name,
            HiscoreValues: hiscoreValues,
            HiscoreData: hiscoreData,
            // Explicitly reference time.Now so that queries can compare against their own time.Now
            CreatedAt: time.Now().Unix(),
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

        extraData := make(map[string]string)
        for key, value := range h.DataMap {
            extraData[key] = value
        }
        delete(extraData, "team")
        delete(extraData, "className")
        delete(extraData, "bountyColor")

        result = append(result, HiscoreEntry {
            Name: h.Hiscore.Name,
            Team: h.DataMap["team"],
            Kills: h.ValueMap["kills"],
            Deaths: h.ValueMap["deaths"],
            ClassName: h.DataMap["className"],
            Bounty: h.ValueMap["bounty"],
            BountyColor: h.DataMap["bountyColor"],
            ExtraValues: extraValues,
            ExtraData: extraData,
        })
    }
    return result
}
