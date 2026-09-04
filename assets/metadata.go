// Package assets stores optional metadata used by asset and layout tooling.
package assets

import (
	"sync"

	"rtgui/core"
)

type AssetInfo struct {
	Name    string
	Size    core.Vec2
	Atlas   core.Rect
	Padding core.Rect
	MinSize core.Vec2
}

type Registry struct {
	mu      sync.RWMutex
	entries map[string]AssetInfo
}

func NewRegistry() *Registry { return &Registry{entries: map[string]AssetInfo{}} }

func (r *Registry) Register(asset AssetInfo) {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.entries == nil {
		r.entries = map[string]AssetInfo{}
	}
	r.entries[asset.Name] = asset
	r.mu.Unlock()
}

func (r *Registry) Get(name string) (AssetInfo, bool) {
	if r == nil {
		return AssetInfo{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	asset, ok := r.entries[name]
	return asset, ok
}

func (r *Registry) Clear() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.entries = map[string]AssetInfo{}
	r.mu.Unlock()
}
