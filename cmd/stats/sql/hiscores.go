package sql

import (
    "errors"
    "gorm.io/gorm"
    "gorm.io/driver/sqlite"
    "sort"
)

//////////////////////////////////////////////////

type HiscoreWithMap struct {
    Hiscore *Hiscore
    ValueMap map[string]int64
}

func (r *Hiscore) withMap() HiscoreWithMap {
    valueMap := make(map[string]int64)
    for _, v := range r.HiscoreValues {
        valueMap[v.Key] = v.Value
    }
    return HiscoreWithMap { Hiscore: r, ValueMap: valueMap }
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

func (hdb *HiscoresDbTransaction) Cull(bottomN int64, key string) (int64, error) {
    result := hdb.db.Exec(
        `
        delete from hiscores where id in (
            select h.id from hiscores h
            inner join hiscore_values hv
            on h.id = hv.hiscore_id
            where hv.key = ?
            order by hv.value asc limit ?
        );
        `, key, bottomN,
    )
    if result.Error != nil {
        return 0, result.Error
    }
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
    result = hdb.db.Preload("HiscoreValues").Find(&results, pks)
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
