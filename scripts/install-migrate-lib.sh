#!/bin/bash

# Must run outside of project folder
go get -tags "sqlite3 postgres" -u github.com/golang-migrate/migrate/cmd/migrate
