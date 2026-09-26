// Package ids holds the key types of entities: proto messages that implement
// [entity.ID], so an entity whose id field takes one needs no hand-written
// code for protoc-gen-go-entity's output to compile.
package ids

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"uuid"

	"github.com/iv-one/x/entity"
	"github.com/iv-one/x/errorsx/raise"
	"github.com/rs/xid"
)

var (
	_ entity.ID = (*ID)(nil)
	_ entity.ID = (*UUID)(nil)
	_ entity.ID = (*TenantID)(nil)
	_ entity.ID = (*SID)(nil)
	_ entity.ID = (*EmailID)(nil)
	_ entity.ID = (*TokenID)(nil)
	_ entity.ID = (*Rel)(nil)
	_ entity.ID = (*NilID)(nil)
)

// ID

// NewID returns a new xid.
func NewID() *ID {
	return &ID{Ids: xid.New().Bytes()}
}

// IDFromString parses the string form of an xid.
func IDFromString(s string) (*ID, error) {
	x, err := xid.FromString(s)
	if err != nil {
		return nil, raise.Error(err)
	}
	return &ID{Ids: x.Bytes()}, nil
}

// Str returns the xid string. It panics when the bytes are not an xid.
func (t *ID) Str() string {
	x, err := xid.FromBytes(t.Ids)
	if err != nil {
		panic(err)
	}
	return x.String()
}

// Bytes returns the xid bytes.
func (t *ID) Bytes() []byte { return t.Ids }

// Eq reports whether id has the same string form.
func (t *ID) Eq(id entity.ID) bool { return id != nil && t.Str() == id.Str() }

// NotEq reports whether id is nil or has a different string form.
func (t *ID) NotEq(id entity.ID) bool { return id == nil || t.Str() != id.Str() }

// IsNil reports whether the id is unset.
func (t *ID) IsNil() bool { return t == nil || t.Ids == nil }

// UUID

// NewUUID returns a new random UUID.
func NewUUID() *UUID {
	id := uuid.New()
	return &UUID{Uuid: id[:]}
}

// UUIDFromString parses the canonical UUID form.
func UUIDFromString(s string) (*UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, raise.Error(err)
	}
	return &UUID{Uuid: id[:]}, nil
}

// Str returns the canonical UUID form.
func (t *UUID) Str() string {
	var id uuid.UUID
	copy(id[:], t.Uuid[:16])
	return id.String()
}

// Bytes returns the 16 UUID bytes.
func (t *UUID) Bytes() []byte { return t.Uuid[:16] }

// Eq reports whether id has the same string form.
func (t *UUID) Eq(id entity.ID) bool { return id != nil && t.Str() == id.Str() }

// IsNil reports whether the id is unset.
func (t *UUID) IsNil() bool { return t == nil || t.Uuid == nil }

// TenantID

// ErrInvalidTenantID is returned for a string that is not a hex uint32.
var ErrInvalidTenantID = errors.New("invalid tenant ID")

// tenantCounter is the last number NewTenantID handed out, seeded at random.
var tenantCounter = randUint24()

// NewTenantID returns the next tenant id of a process-local counter. It is
// unique within the process only; a store hands out lasting tenant ids.
func NewTenantID() *TenantID {
	return TenantIDFromInt32(atomic.AddUint32(&tenantCounter, 1))
}

// TenantIDFromInt32 returns the tenant id of n.
func TenantIDFromInt32(n uint32) *TenantID {
	var id [4]byte
	binary.LittleEndian.PutUint32(id[:], n)
	return &TenantID{Tid: id[:]}
}

// TenantIDFromBytes returns the tenant id of the first four bytes of xs,
// zero-padded when xs is shorter.
func TenantIDFromBytes(xs []byte) *TenantID {
	var id [4]byte
	copy(id[:], xs)
	return &TenantID{Tid: id[:]}
}

// TenantIDFromString parses the hex form, with or without 0x.
func TenantIDFromString(s string) (*TenantID, error) {
	n, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 32)
	if err != nil {
		return nil, ErrInvalidTenantID
	}
	return TenantIDFromInt32(uint32(n)), nil
}

// UInt32 returns the number of the tenant id.
func (t *TenantID) UInt32() uint32 { return binary.LittleEndian.Uint32(t.Tid) }

// Str returns the hex form, 0x1a2b.
func (t *TenantID) Str() string { return fmt.Sprintf("0x%x", t.UInt32()) }

// Bytes returns the four key-prefix bytes.
func (t *TenantID) Bytes() []byte { return t.Tid }

// Eq reports whether other has the same string form.
func (t *TenantID) Eq(other entity.ID) bool { return t.Str() == other.Str() }

// IsNil reports whether the id is unset.
func (t *TenantID) IsNil() bool { return t == nil || t.Tid == nil }

func randUint24() uint32 {
	b := make([]byte, 3)
	if _, err := rand.Reader.Read(b); err != nil {
		panic(raise.Errorf("ids: cannot generate random number: %v;", err))
	}
	return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
}

// SID

// NewSID returns the string id s.
func NewSID(s string) *SID { return &SID{Sid: s} }

// Str returns the string.
func (t *SID) Str() string { return t.Sid }

// Bytes returns the string's bytes.
func (t *SID) Bytes() []byte { return []byte(t.Sid) }

// Eq reports whether id has the same string.
func (t *SID) Eq(id entity.ID) bool { return id != nil && t.Sid == id.Str() }

// IsNil reports whether the id is unset or empty.
func (t *SID) IsNil() bool { return t == nil || t.Sid == "" }

// EmailID

// NewEmailID returns the email id of s, lowercased and trimmed.
func NewEmailID(s string) *EmailID {
	return &EmailID{Email: strings.ToLower(strings.TrimSpace(s))}
}

// Str returns the address.
func (t *EmailID) Str() string { return t.Email }

// Bytes returns the address's bytes.
func (t *EmailID) Bytes() []byte { return []byte(t.Email) }

// Eq reports whether id has the same string form.
func (t *EmailID) Eq(id entity.ID) bool { return id != nil && t.Str() == id.Str() }

// ToSID returns the address as a string id.
func (t *EmailID) ToSID() *SID { return NewSID(t.Str()) }

// IsNil reports whether the id is unset or empty.
func (t *EmailID) IsNil() bool { return t == nil || t.Email == "" }

// TokenID

// ErrInvalidTokenID is returned for a string that is not base64url.
var ErrInvalidTokenID = errors.New("invalid token id")

// NewTokenID returns a token of length random bytes.
func NewTokenID(length int) *TokenID {
	token := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, token); err != nil {
		panic(err)
	}
	return &TokenID{Token: token}
}

// TokenIDFromString parses the base64url form.
func TokenIDFromString(s string) (*TokenID, error) {
	token, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, raise.Errors(ErrInvalidTokenID, err)
	}
	return &TokenID{Token: token}, nil
}

// Str returns the base64url form.
func (t *TokenID) Str() string { return base64.URLEncoding.EncodeToString(t.Token) }

// Bytes returns the token bytes.
func (t *TokenID) Bytes() []byte { return t.Token }

// Eq reports whether id has the same string form.
func (t *TokenID) Eq(id entity.ID) bool { return id != nil && t.Str() == id.Str() }

// IsNil reports whether the id is unset.
func (t *TokenID) IsNil() bool { return t == nil || t.Token == nil }

// Rel

// NewRel returns the relation from one entity to another.
func NewRel(from, to *ID) *Rel { return &Rel{From: from, To: to} }

// Str returns from:to.
func (t *Rel) Str() string { return fmt.Sprintf("%s:%s", t.From.Str(), t.To.Str()) }

// Bytes returns the from bytes followed by the to bytes.
func (t *Rel) Bytes() []byte { return append(t.From.Bytes(), t.To.Bytes()...) }

// Eq reports whether id has the same string form.
func (t *Rel) Eq(id entity.ID) bool { return id != nil && t.Str() == id.Str() }

// IsNil reports whether the relation or either end is unset.
func (t *Rel) IsNil() bool { return t == nil || t.From == nil || t.To == nil }

// NilID

// NID is the empty id.
var NID = &NilID{}

// Str returns "".
func (t *NilID) Str() string { return "" }

// Bytes returns no bytes.
func (t *NilID) Bytes() []byte { return []byte{} }

// Eq reports whether id is nil or empty.
func (t *NilID) Eq(id entity.ID) bool { return id == nil || id.Str() == "" }

// IsNil reports whether t is nil.
func (t *NilID) IsNil() bool { return t == nil }
