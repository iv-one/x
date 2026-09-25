package x

import (
	"uuid"

	"github.com/rs/xid"
)

// GUID generates a new GUID.
func GUID() string {
	guid := xid.New()
	return guid.String()
}

// UUIDv4 generates a new UUIDv4.
func UUIDv4() string {
	return uuid.NewV4().String()
}
