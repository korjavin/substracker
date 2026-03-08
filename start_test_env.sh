#!/bin/bash
export DB_PATH="test_data.db"
export PORT="5454"
rm -f test_data.db
./start.sh &
echo $! > server.pid
