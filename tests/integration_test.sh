#!/bin/bash

echo 'Should be getting: "Auth Failed"'
psql -U postgres -h 127.0.0.1 -p 5432
psql -h 127.0.0.1 -p 5432 -U postgres
psql -h 127.0.0.1 -p 5432 -U postgres 'sslmode=disable'

echo '------------------------------------------'
echo 'Should be getting: server does not support SSL, but SSL was required'
psql -h 127.0.0.1 -p 5432 -U postgres 'sslmode=require'

echo '------------------------------------------'
echo 'Should be getting: "ERROR:  No such user:"'
for x in $(seq 1 50); do
  sleep .2
  a=$(printf "%-${x}s" "a")
  user="${a// /a}"
  psql -U ${user} -h 127.0.0.1 -p 5432
done
