package sql

import (
    "testing"
)

var hdb *HiscoresDb

func TestCull(t *testing.T) {
    tx := hdb.MakeTransaction()
    defer tx.Rollback()

    tx.Insert([]Hiscore {
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
        {
            Name: "Jack",
            HiscoreValues: []HiscoreValue {
                { Key: "Kills", Value: 2 },
                { Key: "Deaths", Value: 9009 },
                { Key: "IQ", Value: 77 },
            },
        },
    })

    eqCulled, err := tx.Cull(2, "EQ")
    if err != nil {
        t.Error(err)
    }
    if eqCulled != 0 {
        t.Fatalf("Expected 0 EQ culls, got %d", eqCulled)
    }
    iqCulled, err := tx.Cull(2, "IQ")
    if err != nil {
        t.Error(err)
    }
    if iqCulled != 2 {
        t.Fatalf("Expected 2 IQ culls, got %d", iqCulled)
    }
    topKills, err := tx.Select(5, "Kills")
    if err != nil {
        t.Error(err)
    }
    if len(topKills) != 1 {
        t.Fatalf("Expected 1 remaining row, got %d", len(topKills))
    }
    remainingIq := topKills[0].Hiscore.withMap().ValueMap["IQ"]
    if remainingIq != 77 {
        t.Fatalf("Expected remaining IQ to be 77, got %d", remainingIq)
    }
}

func TestInsertAndSelectTop(t *testing.T) {
    tx := hdb.MakeTransaction()
    defer tx.Rollback();

    tx.Insert([]Hiscore {
        {
            Name: "Bob",
            HiscoreValues: []HiscoreValue {
                { Key: "Kills", Value: 9001 },
                { Key: "Deaths", Value: 2 },
                { Key: "IQ", Value: 3 },
                { Key: "BobOnlyRecord", Value: 444 },
            },
            HiscoreData: []HiscoreData {
                { Key: "MVP", Value: "MVP" },
            },
        },
        {
            Name: "Jill",
            HiscoreValues: []HiscoreValue {
                { Key: "Kills", Value: 2 },
                { Key: "Deaths", Value: 9009 },
                { Key: "IQ", Value: 3 },
            },
            HiscoreData: []HiscoreData {
                { Key: "Worst Player", Value: "Worst Player" },
            },
        },
    })

    topKills, err := tx.Select(1, "Kills")
    if err != nil {
        t.Error(err)
    }
    if len(topKills) != 1 {
        t.Fatalf("Expected 1 row, got %d", len(topKills))
    }
    if topKills[0].Hiscore.Name != "Bob" {
        t.Fatalf("Expected Bob, got %s", topKills[0].Hiscore.Name)
    }
    valuesLen := len(topKills[0].Hiscore.HiscoreValues)
    if valuesLen != 4 {
        t.Fatalf("Expected 4 values for Bob, got %d", valuesLen)
    }
    bobOnlyRecord := topKills[0].ValueMap["BobOnlyRecord"]
    if bobOnlyRecord != 444 {
        t.Fatalf("Expected BobOnlyRecord to exist and be 444, got %d", bobOnlyRecord)
    }
    mvp := topKills[0].DataMap["MVP"]
    if mvp != "MVP" {
        t.Fatalf("Expected MVP to exist and be MVP, got %s", mvp)
    }

    topDeaths, err := tx.Select(55, "Deaths")
    if err != nil {
        t.Error(err)
    }
    if len(topDeaths) != 2 {
        t.Fatalf("Expected 2 rows, got %d", len(topDeaths))
    }
    if topDeaths[0].Hiscore.Name != "Jill" {
        t.Fatalf("Expected Jill, got %s", topDeaths[0].Hiscore.Name)
    }
    if topDeaths[1].Hiscore.Name != "Bob" {
        t.Fatalf("Expected Bob, got %s", topDeaths[1].Hiscore.Name)
    }
    valuesLen = len(topDeaths[0].Hiscore.HiscoreValues)
    if valuesLen != 3 {
        t.Fatalf("Expected 3 values for Jill, got %d", valuesLen)
    }

    topUnknown, err := tx.Select(5, "Unknown")
    if err != nil {
        t.Error(err)
    }
    if len(topUnknown) != 0 {
        t.Fatalf("Expected 0 rows, got %d", len(topUnknown))
    }
}

func TestMain(m *testing.M) {
    hdb = RemakeTestDb()
    m.Run()
}
