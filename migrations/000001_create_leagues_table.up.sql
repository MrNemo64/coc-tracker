BEGIN;

CREATE TABLE IF NOT EXISTS capital_leagues (
    id INTEGER PRIMARY KEY,
    name VARCHAR
);

CREATE TABLE IF NOT EXISTS player_leagues (
    id INTEGER PRIMARY KEY,
    name VARCHAR
);

CREATE TABLE IF NOT EXISTS builder_base_leagues (
    id INTEGER PRIMARY KEY,
    name VARCHAR
);

CREATE TABLE IF NOT EXISTS war_leagues (
    id INTEGER PRIMARY KEY,
    name VARCHAR
);

COMMIT;