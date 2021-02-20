#!/bin/bash

# Must run outside of project folder
PWD=$(pwd)
cd /
go get -tags "sqlite3" -u github.com/golang-migrate/migrate/cmd/migrate
cd $PWD
