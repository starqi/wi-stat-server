#!/bin/bash

echo -n "Enter filename: "
read

migrate create -ext sql -dir ../db/migrations $REPLY
if [ $? != 0 ]; then
    echo error
    read
    exit 1
fi

echo done
read
