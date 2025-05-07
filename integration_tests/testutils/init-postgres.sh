#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Enable UUID extension
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
    
    -- Enable any other extensions your app might need
    -- CREATE EXTENSION IF NOT EXISTS "hstore";
    -- CREATE EXTENSION IF NOT EXISTS "postgis";
    
    -- Add any other database initialization here
EOSQL
