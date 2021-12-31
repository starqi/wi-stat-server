package main

import (
    "testing"
    "os"
    "net/http"
    "io/ioutil"
    "crypto/x509"
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "encoding/pem"
    "encoding/base64"
    "crypto"
    "strings"
    hsql "github.com/starqi/wi-util-servers/cmd/stats/sql"
)

func TestPostThenGetTop(t *testing.T) {
    json := `
        [
            {"name": "Bill", "team": "Sutasu", "kills": 0, "deaths": 44, "className": "Medic", "bounty": 34, "bountyColor": "red", "extraValues": {"healed": 1}, "extraData": {"mvp": "MVP"}},
            {"name": "Joshy", "team": "Sutasu", "kills": 5, "deaths": 3, "bounty": 34, "bountyColor": "red", "className": "Marine", "extraData": {"mvp": "MVP"}},
            {"name": "Alexandria", "bounty": 34, "bountyColor": "red", "team": "Sutasu", "kills": 11, "deaths": 0, "className": "Marine"}
        ]
    `
    jsonHash := sha256.Sum256([]byte(json))
    sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, jsonHash[:])
    if err != nil {
        t.Fatal("Failed to sign - ", err)
    }
    base64Sig := base64.StdEncoding.EncodeToString(sig)

    jsonReader := strings.NewReader(json)
    req, err := http.NewRequest("POST", "http://localhost:8088/hiscore", jsonReader)
    if err != nil {
        t.Fatal("Failed to make request - ", err)
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set(signatureHeader, base64Sig)

    res, err := http.DefaultClient.Do(req)
    if err != nil {
        t.Fatal("Failed to POST request - ", err)
    }

    bytes, err := ioutil.ReadAll(res.Body)
    if err != nil {
        t.Fatal("Failed to read body - ", err)
    }

    res.Body.Close()
    defer res.Body.Close()

    str := string(bytes)
    t.Log("Body - ", str)

    if res.StatusCode != 200 {
        t.Fatalf("Expected 200 on POST, got %d", res.StatusCode)
    }

    //////////////////////////////////////////////////

    res, err = http.Get("http://localhost:8088/hiscore/top?field=kills&num=2")
    if err != nil {
        t.Fatal("Failed to GET top scores request - ", err)
    }

    bytes, err = ioutil.ReadAll(res.Body)
    if err != nil {
        t.Fatal("Failed to read body - ", err)
    }

    res.Body.Close()
    str = string(bytes)
    t.Log("Body - ", str)

    if res.StatusCode != 200 {
        t.Fatalf("Expected 200 on GET, got %d", res.StatusCode)
    }
}

var key *rsa.PrivateKey
var client *http.Client

func TestMain(m *testing.M) {

    client = &http.Client{}

    _key, err := rsa.GenerateKey(rand.Reader, 4096)
    if err != nil {
        println(err)
        panic("Failed to generate key")
    }
    key = _key

    bytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
    if err != nil {
        println(err)
        panic("Failed to marshal as PKIX")
    }

    publicKeyBytes := []byte(pem.EncodeToMemory(&pem.Block {
        Type: "PUBLIC KEY",
        Bytes: bytes,
    }))
    
    hsql.RemakeTestDb()

    os.Setenv(relativeDbPathEnv, "../../../dist/test-db.db")
    os.Setenv("PORT", "8088")
    os.Setenv(postPublicKeyEnv, string(publicKeyBytes))

    go main()
    m.Run()
}
