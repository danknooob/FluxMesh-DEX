-- List all tables in the current database (fluxmesh).
-- Run with: psql "host=localhost user=fluxmesh password=fluxmesh_secret dbname=fluxmesh port=5432 sslmode=disable" -f scripts/check-db-tables.sql
-- Or set DB_DSN and: psql "$DB_DSN" -f scripts/check-db-tables.sql

\echo '=== Tables in database: fluxmesh (or current DB) ==='
SELECT schemaname, tablename
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY tablename;

\echo ''
\echo '=== Expected tables ==='
\echo '  API creates:     users, orders, markets, balances'
\echo '  Indexer creates: trades'
\echo '  (Stored procs/functions are created by API migrations)'
