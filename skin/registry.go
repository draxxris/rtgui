package skin

import (
	"sync"

	"rtgui/core"
)

type Registry struct {
	mu      sync.RWMutex
	entries map[SkinKey]SkinDescriptor
}

func NewRegistry() *Registry { return &Registry{entries: map[SkinKey]SkinDescriptor{}} }

func (r *Registry) Set(key SkinKey, descriptor SkinDescriptor) {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.entries == nil {
		r.entries = map[SkinKey]SkinDescriptor{}
	}
	r.entries[key] = descriptor
	r.mu.Unlock()
}

func (r *Registry) Get(key SkinKey) (SkinDescriptor, bool) {
	if r == nil {
		return SkinDescriptor{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	descriptor, ok := r.entries[key]
	return descriptor, ok
}

// Lookup tries the requested state and then falls back to the normal state.
func (r *Registry) Lookup(kind core.WidgetKind, part SkinPart, state core.WidgetState) (SkinDescriptor, bool) {
	if r == nil {
		return SkinDescriptor{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
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

func (r *Registry) Clear() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.entries = map[SkinKey]SkinDescriptor{}
	r.mu.Unlock()
}
