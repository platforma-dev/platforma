-- +migrate Up
CREATE TABLE IF NOT EXISTS sessions (
	id VARCHAR(255) PRIMARY KEY,
	"user" VARCHAR(255),
	created TIMESTAMP,
	expires TIMESTAMP
);

-- +migrate Down
DROP TABLE sessions;
