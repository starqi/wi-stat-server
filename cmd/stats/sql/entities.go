package sql

type Hiscore struct {
    ID int64
    Name string
    HiscoreValues []HiscoreValue
    CreatedAt uint64
}

type HiscoreValue struct {
    ID int64
    HiscoreID int64
    Key string
    Value int64
}
