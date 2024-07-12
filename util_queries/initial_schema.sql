CREATE TABLE IF NOT EXISTS migrations (
    migration_number INTEGER,
    migration_time TIMESTAMP default NOW()
);
