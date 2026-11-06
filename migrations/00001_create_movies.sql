
-- +goose Up
CREATE TABLE movies (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    year INTEGER NOT NULL,
    director TEXT NOT NULL
);

-- +goose Down
DROP TABLE movies;
