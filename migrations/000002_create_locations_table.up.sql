BEGIN;

CREATE TABLE IF NOT EXISTS locations (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL,
    is_country BOOLEAN NOT NULL,
    country_code VARCHAR
);

COMMIT;