#!/bin/bash

mkdir -p ../dist
migrate -source file://../db/migrations -database sqlite3://../dist/db.db up
if [ $? != 0 ]; then
    echo error
    read
    exit 1
fi

echo finished
read
