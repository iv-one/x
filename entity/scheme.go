package entity

// Meta is what storage needs to know about one entity type.
type Meta struct {
	// Namespace prefixes every key of the type; zero means unregistered.
	Namespace uint16
	// Tenant scopes the type's keys to the tenant in the context.
	Tenant bool
	// Sensitive encrypts the type at rest.
	Sensitive bool
}

// Scheme maps each entity type to its Meta. protoc-gen-go-entity generates one
// from the options in options.proto.
type Scheme map[EntityType]Meta

// Namespace returns the key prefix of kind, or zero when kind is unregistered.
func (s Scheme) Namespace(kind EntityType) uint16 { return s[kind].Namespace }

// IsTenantBase reports whether kind's keys are scoped to a tenant.
func (s Scheme) IsTenantBase(kind EntityType) bool { return s[kind].Tenant }

// IsSensitive reports whether kind is encrypted at rest.
func (s Scheme) IsSensitive(kind EntityType) bool { return s[kind].Sensitive }
