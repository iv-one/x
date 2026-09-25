// Package imagex decides whether user-supplied bytes may become an asset, and
// produces the canonical bytes stored when they may.
//
// Meant to be the only code that inspects an upload before it is written
// anywhere. Takes an io.Reader and returns bytes or
// a typed rejection, with no HTTP, storage or configuration in the way.
//
// The order of checks is load-bearing; see Canonical.
package imagex
