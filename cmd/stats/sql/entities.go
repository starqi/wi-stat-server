package sql

type Hiscore struct {
    ID int64
    Name string
    HiscoreValues []HiscoreValue
    HiscoreData []HiscoreData
    CreatedAt uint64
}

type HiscoreValue struct {
    ID int64
    HiscoreID int64
    Key string
    Value int64
}

type HiscoreData struct {
    ID int64
    HiscoreID int64
    Key string
    Value string
}
