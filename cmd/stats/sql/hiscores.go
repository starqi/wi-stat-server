package sql

import (
    "time"
    "errors"
    "log"
    "gorm.io/gorm"
    "gorm.io/driver/sqlite"
    "sort"
)

//////////////////////////////////////////////////

type HiscoreWithMap struct {
    Hiscore *Hiscore
    ValueMap map[string]int64
    DataMap map[string]string
}

func (r *Hiscore) withMap() HiscoreWithMap {
    valueMap := make(map[string]int64)
    for _, v := range r.HiscoreValues {
        valueMap[v.Key] = v.Value
    }

    dataMap := make(map[string]string)
    for _, v := range r.HiscoreData {
        dataMap[v.Key] = v.Value
    }

    return HiscoreWithMap { Hiscore: r, ValueMap: valueMap, DataMap: dataMap }
}

//////////////////////////////////////////////////

type MaxSortedHiscores struct {
    rows []HiscoreWithMap
    key string
}

func (r *MaxSortedHiscores) Len() int {
    return len(r.rows)
}

func (r *MaxSortedHiscores) Less(i, j int) bool {
    return r.rows[i].ValueMap[r.key] > r.rows[j].ValueMap[r.key]

}

func (r *MaxSortedHiscores) Swap(i, j int) {
    t := r.rows[i]
    r.rows[i] = r.rows[j]
    r.rows[j] = t
}

//////////////////////////////////////////////////

const secondsPerDay = 24 * 3600
var timeGroupSeconds = [3]int64{ 7 * secondsPerDay, 30 * secondsPerDay, 0 }

type HiscoresDbTransaction struct {
    hdb *HiscoresDb
    db *gorm.DB
}

type HiscoresDb struct {
    db *gorm.DB
}

func MakeHiscoresDb(sqliteDbPath string) (*HiscoresDb, error) {
    db, err := gorm.Open(sqlite.Open(sqliteDbPath), &gorm.Config{})
    if err != nil {
        return nil, err
    }
    return &HiscoresDb { db }, nil
}

func (hdb *HiscoresDb) MakeTransaction() HiscoresDbTransaction {
    return HiscoresDbTransaction { hdb: hdb, db: hdb.db.Begin() }
}

func (hdb *HiscoresDbTransaction) Rollback() {
    hdb.db.Rollback()
}

func (hdb *HiscoresDbTransaction) Commit() {
    hdb.db.Commit()
}

func (hdb *HiscoresDb) Transaction(do func (tx *HiscoresDbTransaction) (interface{}, error)) (interface{}, error) {
    tx := hdb.MakeTransaction()
    result, err := do(&tx)
    if err != nil {
        tx.Rollback()
    } else {
        tx.Commit()
    }
    return result, err
}

// TODO More tests for time groups
func (hdb *HiscoresDbTransaction) Cull(topNToKeep int, columns []string) (int64, error) {
    if len(columns) <= 0 {
        return 0, errors.New("Column count must be > 0")
    }

    log.Printf("Starting cull for top %d, columns %v", topNToKeep, columns)

    now := time.Now().Unix()
    pks := make([]int64, 0)
    for _, seconds := range timeGroupSeconds {
        var minSeconds int64
        if seconds <= 0 {
            minSeconds = 0
        } else {
            minSeconds = now - seconds
        }

        for _, column := range columns {
            _pks, err := hdb.getTopPks(topNToKeep, column, minSeconds)
            if err != nil { return 0, err }
            for _, _pk := range _pks {
                pks = append(pks, _pk)
            }
        }
    }

    result := hdb.db.Exec("delete from hiscores where id not in ?", pks)
    if result.Error != nil {
        return 0, result.Error
    }

    log.Printf("Culled %d rows", result.RowsAffected)
    return result.RowsAffected, nil
}

// TODO Check which is first: limit or distinct
func (hdb *HiscoresDbTransaction) Select(topN int, key string, minSeconds int64) ([]HiscoreWithMap, error) {
    pks, err := hdb.getTopPks(topN, key, minSeconds)
    if err != nil { return nil, err }
    if len(pks) == 0 { return []HiscoreWithMap{}, nil }

    var hiscores []Hiscore
    hiscoresResult := hdb.db.Preload("HiscoreData").Preload("HiscoreValues").Find(&hiscores, pks)
    if hiscoresResult.Error != nil { return nil, hiscoresResult.Error }
    if len(hiscores) == 0 { return nil, errors.New("Unexpected: Zero results but initially not zero") }

    hiscores2 := make([]HiscoreWithMap, 0, len(hiscores))
    for i := range hiscores {
        hiscores2 = append(hiscores2, hiscores[i].withMap())
    }

    sort.Sort(&MaxSortedHiscores { rows: hiscores2, key: key })
    return hiscores2, nil
}

func (hdb *HiscoresDbTransaction) Insert(entries []Hiscore) (int64, error) {
    result := hdb.db.Create(entries)
    if result.Error != nil {
        return 0, result.Error
    }
    return result.RowsAffected, nil
}

func (hdb *HiscoresDbTransaction) getTopPks(topN int, key string, minMs int64) ([]int64, error) {
    var pks []int64
    result := hdb.db.Raw(`
        select h.id from hiscores h
        inner join hiscore_values hv 
        on h.id = hv.hiscore_id
        where hv.key = ? and hv.value in (
            select distinct hv.value from hiscores h
            inner join hiscore_values hv
            on h.id = hv.hiscore_id
            where hv.key = ? and h.created_at > ?
            order by hv.value desc limit ?
        ) and h.created_at > ?
    `, key, key, minMs, topN, minMs).Scan(&pks)

    if result.Error != nil {
        return nil, result.Error
    }

    return pks, nil
}
