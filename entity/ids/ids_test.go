package ids

import (
	"testing"
	"uuid"

	"github.com/iv-one/x/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tid(b [4]byte) *TenantID {
	return TenantIDFromBytes(b[:])
}

func TestFromInt32(t *testing.T) {
	assert.Equal(t, tid([4]byte{0, 0, 0, 0}), TenantIDFromInt32(0))
	assert.Equal(t, tid([4]byte{2, 0, 0, 0}), TenantIDFromInt32(2))
	assert.Equal(t, tid([4]byte{0xff, 0xff, 0xff, 0xff}), TenantIDFromInt32(^uint32(0)))
	assert.Equal(t, tid([4]byte{0xd2, 0x02, 0x96, 0x49}), TenantIDFromInt32(1234567890))
}

func TestTenantID_UInt32(t *testing.T) {
	assert.Equal(t, uint32(0), tid([4]byte{0, 0, 0, 0}).UInt32())
	assert.Equal(t, uint32(1234567890), tid([4]byte{0xd2, 0x02, 0x96, 0x49}).UInt32())
}

func TestTenantID_Str(t *testing.T) {
	assert.Equal(t, "0x0", tid([4]byte{0, 0, 0, 0}).Str())
	assert.Equal(t, "0x499602d2", tid([4]byte{0xd2, 0x02, 0x96, 0x49}).Str())
}

func TestFromString(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		id, err := TenantIDFromString("0x499602d2")
		require.NoError(t, err)
		assert.Equal(t, tid([4]byte{0xd2, 0x02, 0x96, 0x49}), id)
	})

	t.Run("invalid input", func(t *testing.T) {
		_, err := TenantIDFromString("invalid")
		require.Error(t, err)
		assert.Equal(t, ErrInvalidTenantID, err)
	})
}

func TestTenantIDFromBytes(t *testing.T) {
	assert.Equal(t, tid([4]byte{0xd2, 0x02, 0x96, 0x49}), TenantIDFromBytes([]byte{0xd2, 0x02, 0x96, 0x49}))
	assert.Equal(t, tid([4]byte{0xd2, 0x02, 0x96, 0x49}), TenantIDFromBytes([]byte{0xd2, 0x02, 0x96, 0x49, 0x51}))
	assert.Equal(t, tid([4]byte{0xd2, 0x00, 0x00, 0x00}), TenantIDFromBytes([]byte{0xd2}))
}

func TestUUID(t *testing.T) {
	id := NewUUID()
	assert.Len(t, id.Bytes(), 16)
	assert.Len(t, id.Str(), 36)

	u, err := uuid.Parse(id.Str())
	require.NoError(t, err)
	assert.Equal(t, id.Bytes(), u[:])
}

func TestNewEmailID(t *testing.T) {
	for _, c := range []string{"mail@mail.com", "MaIl@mail.COM", " mail@mail.com "} {
		assert.Equal(t, "mail@mail.com", NewEmailID(c).Str())
	}
}

// every returns one set id of each type, and every unset form of each.
func every() (set, unset []entity.ID) {
	set = []entity.ID{
		NewID(), NewUUID(), TenantIDFromInt32(7), NewSID("s"), NewEmailID("a@b.c"),
		NewTokenID(16), NewRel(NewID(), NewID()),
	}
	unset = []entity.ID{
		nil, NID, entity.NilID, entity.RawID(nil),
		(*ID)(nil), &ID{}, &ID{Ids: []byte{1, 2}},
		(*UUID)(nil), &UUID{}, &UUID{Uuid: []byte{1}},
		(*TenantID)(nil), &TenantID{}, &TenantID{Tid: []byte{1}},
		(*SID)(nil), &SID{}, (*EmailID)(nil), &EmailID{}, (*TokenID)(nil), &TokenID{},
		(*Rel)(nil), &Rel{}, &Rel{From: NewID()}, (*NilID)(nil),
	}
	return set, unset
}

func TestNeverPanics(t *testing.T) {
	set, unset := every()
	all := append(append([]entity.ID{}, set...), unset...)
	for _, a := range all {
		if a == nil {
			continue
		}
		assert.NotPanics(t, func() {
			_ = a.Str()
			_ = a.Bytes()
			_ = a.IsNil()
			for _, b := range all {
				_ = a.Eq(b)
			}
		}, "%T", a)
	}
}

func TestEq(t *testing.T) {
	set, unset := every()
	for _, a := range set {
		assert.True(t, a.Eq(a), "%T equals itself", a)
		for _, b := range unset {
			assert.False(t, a.Eq(b), "%T set vs %T unset", a, b)
			if b != nil {
				assert.False(t, b.Eq(a), "%T unset vs %T set", b, a)
			}
		}
		for _, b := range set {
			assert.Equal(t, a.Eq(b), b.Eq(a), "%T and %T symmetric", a, b)
		}
	}
	for _, a := range unset {
		if a == nil {
			continue
		}
		for _, b := range unset {
			assert.False(t, a.Eq(b), "%T and %T unset match nothing", a, b)
		}
	}
	assert.True(t, NewSID("x").Eq(NewSID("x")))
	assert.False(t, NewSID("x").Eq(NewSID("y")))
	tok := NewTokenID(16)
	assert.True(t, tok.Eq(&TokenID{Token: append([]byte{}, tok.Token...)}))
}

func TestRelBytesSharesNoMemory(t *testing.T) {
	from := &ID{Ids: make([]byte, 12, 64)}
	to := &ID{Ids: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
	b := NewRel(from, to).Bytes()
	b[0] = 0xff
	assert.Equal(t, byte(0), from.Ids[0])
	assert.Len(t, from.Ids, 12)
	assert.Equal(t, append(make([]byte, 12), to.Ids...)[1:], b[1:])
}
