package entity

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeEntity struct {
	id RawID
}

func (f *fakeEntity) ID() ID                   { return f.id }
func (f *fakeEntity) Type() EntityType         { return "fake" }
func (f *fakeEntity) New() Entity              { return &fakeEntity{} }
func (f *fakeEntity) IsNil() bool              { return f == nil }
func (f *fakeEntity) Update(TimeMachine)       {}
func (f *fakeEntity) Marshal() ([]byte, error) { return f.id, nil }
func (f *fakeEntity) Unmarshal(b []byte) error { f.id = NewRawID(b); return nil }

func TestEntityTypeString(t *testing.T) {
	assert.Equal(t, "user", EntityType("user").String())
}

func TestNilID(t *testing.T) {
	assert.Empty(t, NilID.Str())
	assert.Equal(t, []byte{}, NilID.Bytes())
	assert.False(t, NilID.IsNil(), "NilID is a non-nil pointer")
	assert.False(t, NilID.Eq(NilID))
	assert.False(t, NilID.Eq(RawID("a")))

	var typedNil *nilID
	assert.True(t, typedNil.IsNil())
}

func TestRawID(t *testing.T) {
	id := RawID("abc")
	assert.Equal(t, "abc", id.Str())
	assert.Equal(t, []byte("abc"), id.Bytes())
	assert.False(t, id.IsNil())

	assert.True(t, id.Eq(RawID("abc")))
	assert.False(t, id.Eq(RawID("abd")))
	assert.False(t, id.Eq(nil))
	assert.False(t, id.Eq(RawID(nil)), "an unset id equals nothing")
	assert.False(t, RawID(nil).Eq(RawID(nil)), "two missing ids never match")
	assert.True(t, RawID(nil).IsNil())
}

func TestNewRawIDCopies(t *testing.T) {
	src := []byte("abc")
	id := NewRawID(src)
	src[0] = 'z'
	assert.Equal(t, "abc", id.Str())
}

func TestNewRandomRawID(t *testing.T) {
	id := NewRandomRawID()
	require.Len(t, id, 4)
	assert.False(t, id.IsNil())
}

func TestEList(t *testing.T) {
	a, b := &fakeEntity{id: RawID("a")}, &fakeEntity{id: RawID("b")}
	l := EList([]*fakeEntity{a, b})
	assert.False(t, l.IsNil())
	assert.Equal(t, []Entity{a, b}, l.List())

	empty := EList[*fakeEntity](nil)
	assert.False(t, empty.IsNil(), "a nil slice becomes an empty list")
	assert.Empty(t, empty.List())

	var nilList *elist
	assert.True(t, nilList.IsNil())
	assert.True(t, (&elist{}).IsNil())
}

func TestMapConversions(t *testing.T) {
	a, b := &fakeEntity{id: RawID("a")}, &fakeEntity{id: RawID("b")}

	m := ToMap(a, b)
	assert.Equal(t, map[string]*fakeEntity{"a": a, "b": b}, m)

	fromMap := FromMap(m).List()
	assert.ElementsMatch(t, []Entity{a, b}, fromMap)

	idm := map[string]RawID{"a": RawID("a"), "b": RawID("b")}
	idx := FromMapIDX(idm)
	sort.Slice(idx, func(i, j int) bool { return idx[i].Str() < idx[j].Str() })
	assert.Equal(t, []RawID{RawID("a"), RawID("b")}, idx)

	assert.ElementsMatch(t, []ID{RawID("a"), RawID("b")}, FromMapID(idm))
	assert.Equal(t, []ID{RawID("a"), RawID("b")}, ToList([]RawID{RawID("a"), RawID("b")}))
	assert.Empty(t, ToList[RawID](nil))
}

func TestScheme(t *testing.T) {
	s := Scheme{
		"user":   {Namespace: 7, Tenant: true, Sensitive: true},
		"public": {Namespace: 9},
	}
	assert.Equal(t, uint16(7), s.Namespace("user"))
	assert.True(t, s.IsTenantBase("user"))
	assert.True(t, s.IsSensitive("user"))

	assert.Equal(t, uint16(9), s.Namespace("public"))
	assert.False(t, s.IsTenantBase("public"))
	assert.False(t, s.IsSensitive("public"))

	assert.Zero(t, s.Namespace("unknown"), "unregistered types have no namespace")
	assert.False(t, s.IsTenantBase("unknown"))
}
