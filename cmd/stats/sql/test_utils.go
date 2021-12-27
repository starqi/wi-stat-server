package sql

import (
    "os"
    "log"
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/sqlite3"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func RemakeTestDb() *HiscoresDb {
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

    hdb, err := MakeHiscoresDb(testDbPath)
    if err != nil {
        log.Fatal("Could not access DB - ", err)
    }
    return hdb
}
