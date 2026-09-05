package skin

import "github.com/draxxris/rtgui/core"

// Registry is a single-owner descriptor map with exact and normal fallback lookup.
type Registry struct {
	entries map[SkinKey]SkinDescriptor
}

// NewRegistry returns an empty descriptor registry.
func NewRegistry() *Registry { return &Registry{entries: map[SkinKey]SkinDescriptor{}} }

// Set stores descriptor under an exact key.
func (r *Registry) Set(key SkinKey, descriptor SkinDescriptor) {
	if r == nil {
		return
	}
	if r.entries == nil {
		r.entries = map[SkinKey]SkinDescriptor{}
	}
	r.entries[key] = descriptor
}

// Get returns only an exact-key descriptor.
func (r *Registry) Get(key SkinKey) (SkinDescriptor, bool) {
	if r == nil {
		return SkinDescriptor{}, false
	}
	descriptor, ok := r.entries[key]
	return descriptor, ok
}

// Lookup tries the requested state and then its normal-state fallback.
func (r *Registry) Lookup(kind core.WidgetKind, part SkinPart, state core.WidgetState) (SkinDescriptor, bool) {
	if r == nil {
		return SkinDescriptor{}, false
	}
	key := SkinKey{Widget: kind, Part: part, State: state}
	if descriptor, ok := r.entries[key]; ok {
		return descriptor, true
	}
	if state != core.StateNormal {
		key.State = core.StateNormal
		if descriptor, ok := r.entries[key]; ok {
			return descriptor, true
		}
	}
	return SkinDescriptor{}, false
}

// Replace atomically replaces this registry's complete map with a copy of other.
func (r *Registry) Replace(other *Registry) {
	if r == nil {
		return
	}
	entries := make(map[SkinKey]SkinDescriptor)
	if other != nil {
		entries = make(map[SkinKey]SkinDescriptor, len(other.entries))
		for key, descriptor := range other.entries {
			entries[key] = descriptor
		}
	}
	r.entries = entries
}

// Clear removes every descriptor.
func (r *Registry) Clear() {
	if r != nil {
		r.entries = map[SkinKey]SkinDescriptor{}
	}
}
