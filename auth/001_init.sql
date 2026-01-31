-- +migrate Up
CREATE TABLE IF NOT EXISTS users (
	id VARCHAR(255) PRIMARY KEY,
	username VARCHAR(255) UNIQUE,
	password TEXT,
	salt TEXT,
	created TIMESTAMP,
	updated TIMESTAMP,
	status VARCHAR(50)
);

-- +migrate Down
DROP TABLE users;
