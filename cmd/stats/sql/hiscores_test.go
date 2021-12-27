package sql

import (
     "log"
    "testing"
    "os"
    migrate "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/sqlite3"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

var hdb *HiscoresDb

func TestInsertAndSelectTop(t *testing.T) {
    hdb.Insert([]Hiscore {
        {
            Name: "Bob",
            HiscoreValues: []HiscoreValue {
                { Key: "Kills", Value: 9001 },
                { Key: "Deaths", Value: 2 },
                { Key: "IQ", Value: 3 },
                { Key: "BobOnlyRecord", Value: 444 },
            },
        },
        {
            Name: "Jill",
            HiscoreValues: []HiscoreValue {
                { Key: "Kills", Value: 2 },
                { Key: "Deaths", Value: 9009 },
                { Key: "IQ", Value: 3 },
            },
        },
    })

    topKills, err := hdb.Select(1, "Kills")
    if err != nil {
        t.Error(err)
    }
    if len(topKills) != 1 {
        t.Fatalf("Expected 1 row, got %d", len(topKills))
    }
    if topKills[0].hiscore.Name != "Bob" {
        t.Fatalf("Expected Bob, got %s", topKills[0].hiscore.Name)
    }
    valuesLen := len(topKills[0].hiscore.HiscoreValues)
    if valuesLen != 4 {
        t.Fatalf("Expected 4 values for Bob, got %d", valuesLen)
    }
    if topKills[0].valueMap["BobOnlyRecord"] != 444 {
        t.Fatalf("Expected BobOnlyRecord to exist and be 444")
    }

    topDeaths, err := hdb.Select(55, "Deaths")
    if err != nil {
        t.Error(err)
    }
    if len(topDeaths) != 2 {
        t.Fatalf("Expected 2 rows, got %d", len(topDeaths))
    }
    if topDeaths[0].hiscore.Name != "Jill" {
        t.Fatalf("Expected Jill, got %s", topDeaths[0].hiscore.Name)
    }
    if topDeaths[1].hiscore.Name != "Bob" {
        t.Fatalf("Expected Bob, got %s", topDeaths[1].hiscore.Name)
    }
    valuesLen = len(topDeaths[0].hiscore.HiscoreValues)
    if valuesLen != 3 {
        t.Fatalf("Expected 3 values for Jill, got %d", valuesLen)
    }

    topUnknown, err := hdb.Select(5, "Unknown")
    if err != nil {
        t.Error(err)
    }
    if len(topUnknown) != 0 {
        t.Fatalf("Expected 0 rows, got %d", len(topUnknown))
    }
}

func TestMain(m *testing.M) {
    const testDbPath = "../../../dist/test-db.db"
    err := os.Remove(testDbPath)
    if err != nil {
        log.Print("Failed to remove existing DB - ", err)
    }

    migrate, err := migrate.New("file://../../../db/stats-migrations", "sqlite3://" + testDbPath)
    if err != nil {
        log.Fatal("Failed to create migration class - ", err)
    }

    err = migrate.Up()
    if err != nil {
        log.Fatal("Failed to create test DB - ", err)
    }

    _hdb, err := MakeHiscoresDb(testDbPath)
    if err != nil {
        log.Fatal("Could not access DB - ", err)
    }
    hdb = _hdb

    m.Run()
}
