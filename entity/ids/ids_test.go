package ids

import (
	"testing"
	"uuid"

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
