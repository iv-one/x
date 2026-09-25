// Package entity defines the interfaces a stored record implements ([Entity],
// [ID], [EntityList]) and the [Scheme] storage keys them by. Its options.proto
// annotates proto messages with their storage options; protoc-gen-go-entity
// generates the methods and the Scheme from them.
package entity
