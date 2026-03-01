package auth

import "time"

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusDeleted  Status = "deleted"
)

type User struct {
	ID       string    `db:"id"       json:"id"`
	Username string    `db:"username" json:"username"`
	Password string    `db:"password" json:"password"` //nolint:gosec // Not exposed via handlers
	Salt     string    `db:"salt"     json:"salt"`
	Created  time.Time `db:"created"  json:"created"`
	Updated  time.Time `db:"updated"  json:"updated"`
	Status   Status    `db:"status"   json:"status"`
}
