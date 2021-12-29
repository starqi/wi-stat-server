package sql

import (
    "strings"
    "errors"
    "log"
    "gorm.io/gorm"
    "gorm.io/driver/sqlite"
    "sort"
    "strconv"
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

func (hdb *HiscoresDbTransaction) Cull(topNToKeep int64, columns []string) (int64, error) {

    if len(columns) <= 0 {
        return 0, errors.New("Column count must be > 0")
    }

    log.Printf("Starting cull for top %d, columns %v", topNToKeep, columns)

    const withFragmentTemplate = ` as (select h.id from hiscores h
        inner join hiscore_values hv 
        on h.id = hv.hiscore_id
        where hv.key = ? and hv.value in (
            select distinct hv.value from hiscores h
            inner join hiscore_values hv
            on h.id = hv.hiscore_id
            where hv.key = ? order by hv.value desc limit ?
        ))`

    var b strings.Builder
    b.WriteString("delete from hiscores where id not in (with ")
    for i := range columns {
        b.WriteString("col")
        b.WriteString(strconv.Itoa(i))
        b.WriteString(withFragmentTemplate)
        if i < len(columns) - 1 {
            b.WriteString(",")
            b.WriteString("\n")
        }
    }
    b.WriteString("\n")
    for i := range columns {
        b.WriteString("select id from col")
        b.WriteString(strconv.Itoa(i))
        if i < len(columns) - 1 {
            b.WriteString(" union\n")
        }
    }
    b.WriteString("\n)")
    //log.Print("[Cull Query Debug]\n", b.String())

    params := make([]interface{}, 0, len(columns) * 3)
    for _, c := range columns {
        params = append(params, c, c, topNToKeep)
    }

    result := hdb.db.Exec(b.String(), params...)
    if result.Error != nil {
        return 0, result.Error
    }

    log.Printf("Culled %d rows", result.RowsAffected)
    return result.RowsAffected, nil
}

func (hdb *HiscoresDbTransaction) Select(topN int, key string) ([]HiscoreWithMap, error) {
    var pks []int64
    result := hdb.db.Raw(`
        select h.id from hiscores h
        inner join hiscore_values hv
        on h.id = hv.hiscore_id
        where hv.key = ?
        order by hv.value desc limit ?
    `, key, topN).Scan(&pks)
    if result.Error != nil {
        return nil, result.Error
    }
    if len(pks) == 0 {
        return []HiscoreWithMap{}, nil
    }

    var results []Hiscore
    result = hdb.db.Preload("HiscoreData").Preload("HiscoreValues").Find(&results, pks)
    if result.Error != nil {
        return nil, result.Error
    }
    if len(results) == 0 {
        return nil, errors.New("Unexpected: Zero results but initially not zero")
    }

    results2 := make([]HiscoreWithMap, 0, len(results))
    for i := range results {
        results2 = append(results2, results[i].withMap())
    }

    sort.Sort(&MaxSortedHiscores { rows: results2, key: key })

    return results2, nil
}

func (hdb *HiscoresDbTransaction) Insert(entries []Hiscore) (int64, error) {
    result := hdb.db.Create(entries)
    if result.Error != nil {
        return 0, result.Error
    }
    return result.RowsAffected, nil
}
