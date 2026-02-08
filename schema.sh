#!/bin/bash

docker compose exec -T postgres \
  pg_dump \
    --schema-only \
    --no-owner \
    --no-privileges \
    --clean \
    --if-exists \
    -U postgres \
    ipphone_db \
  > db/structure.sql
