package main

import (
    "testing"
    "os"
    "net/http"
    "io/ioutil"
    hsql "github.com/starqi/wi-util-servers/cmd/stats/sql"
)

func TestAbc2(t *testing.T) {
    t.Log("Heyo")
}

func TestAbc(t *testing.T) {
    res, err := http.Get("http://localhost:8088/hiscore/top")
    if err != nil {
        t.Fatal("Failed to GET top scores request - ", err)
    }
    bytes, err := ioutil.ReadAll(res.Body)
    if err != nil {
        t.Fatal("Failed to read body - ", err)
    }
    str := string(bytes)
    t.Log(str)
}

func TestMain(m *testing.M) {
    hdb := hsql.RemakeTestDb()
    hdb.Transaction(func (tx *hsql.HiscoresDbTransaction) (interface{}, error) {
        tx.Insert([]hsql.Hiscore {
            {
                Name: "Bob",
                HiscoreValues: []hsql.HiscoreValue {
                    { Key: "Kills", Value: 9001 },
                    { Key: "Deaths", Value: 2 },
                    { Key: "IQ", Value: 3 },
                    { Key: "BobOnlyRecord", Value: 444 },
                },
                HiscoreData: []hsql.HiscoreData {
                    { Key: "MVP", Value: "MVP" },
                },
            },
            {
                Name: "Jill",
                HiscoreValues: []hsql.HiscoreValue {
                    { Key: "Kills", Value: 2 },
                    { Key: "Deaths", Value: 9009 },
                    { Key: "IQ", Value: 3 },
                },
                HiscoreData: []hsql.HiscoreData {
                    { Key: "Worst Player", Value: "Worst Player" },
                },
            },
        })
        return nil, nil
    })
    os.Setenv(relativeDbPathEnv, "../../../dist/test-db.db")
    os.Setenv("PORT", "8088")
    go main()
    m.Run()
}
