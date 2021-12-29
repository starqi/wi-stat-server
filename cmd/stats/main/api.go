package main

import (
    "time"
    "bytes"
    "net/http"
    "io/ioutil"
    "log"
    "github.com/gin-gonic/gin"
    "os"
    "crypto/rsa"
    "crypto/x509"
    "crypto/sha256"
    "encoding/pem"
    "encoding/base64"
    "crypto"
    "strconv"
    hsql "github.com/starqi/wi-util-servers/cmd/stats/sql"
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
const signatureHeader = "x-json-signature"
const relativeDbPathEnv = "relativeDbPath"
const postPublicKeyEnv = "postPublicKey"

var cullColumns = []string{ "kills", "healed", "bounty" }
var hdb *hsql.HiscoresDb
var cullTicker *time.Ticker
var postPublicKey *rsa.PublicKey

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

    postPublicKeyInput := os.Getenv(postPublicKeyEnv)
    if postPublicKeyInput == "" {
        log.Printf("Missing %s, will not be able to update", postPublicKeyEnv)
    } else {
        _postPublicKeyIsolated, _ := pem.Decode([]byte(postPublicKeyInput))
        _postPublicKeyParsed, err := x509.ParsePKCS1PublicKey(_postPublicKeyIsolated.Bytes)
        if err != nil {
            log.Print("Failed to parse POST public key, will not be able to update, ", postPublicKeyInput, ", ", err)
        } else {
            postPublicKey = _postPublicKeyParsed
            log.Print("Parsed POST public key - ", postPublicKeyInput)
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

    authGroup := router.Group("/hiscore", checkSignature)
    authGroup.POST("", postHiscore)

    router.GET("hiscore/top", getTopHiscores)
    router.Run() // Will use PORT env var
}

func checkSignature(c * gin.Context) {
    if postPublicKey == nil {
        c.AbortWithStatus(http.StatusUnauthorized)
    } else {
        rawData, err := ioutil.ReadAll(c.Request.Body)
        if err != nil {
            log.Print("Failed to get body, ", c.ClientIP())
            c.AbortWithStatus(http.StatusInternalServerError)
            return
        }
        // Must manually ensure body reader is not stuck at EOF 
        c.Request.Body = ioutil.NopCloser(bytes.NewReader(rawData))

        hash := sha256.Sum256(rawData)
        sigBase64 := c.GetHeader(signatureHeader)
        sigBytes, err := base64.StdEncoding.DecodeString(sigBase64)
        if err != nil {
            log.Print("Failed to read base64 signature, ", err)
            c.AbortWithStatus(http.StatusInternalServerError)
        }

        err = rsa.VerifyPKCS1v15(postPublicKey, crypto.SHA256, hash[:], sigBytes)
        if err != nil {
            log.Print("Failed to verify signature, ", c.ClientIP())
            c.AbortWithStatus(http.StatusUnauthorized)
        }
    }

    c.Next()
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

    result, err := hdb.Transaction(func (tx *hsql.HiscoresDbTransaction) (interface{}, error) {
        return tx.Select(num, field)
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
