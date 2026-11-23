// Package errorsx is the error model of AIP-193: an [Error] holds a canonical
// [Code], a stable reason, metadata and a message for a person. It can also
// hold field violations and a retry delay. The HTTP status of an error comes
// from its code. [From] gives the Error of each error.
package errorsx
