-- +migrate Up
CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	email TEXT UNIQUE NOT NULL
);

-- +migrate Down
DROP TABLE users;
