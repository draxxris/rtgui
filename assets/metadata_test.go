package assets

import (
	"testing"

	"rtgui/core"
)

func TestRegistry(t *testing.T) {
	registry := NewRegistry()
	asset := AssetInfo{Name: "button", Size: core.Vec2{X: 64, Y: 32}}
	registry.Register(asset)
	if got, ok := registry.Get("button"); !ok || got != asset {
		t.Fatalf("asset=%+v ok=%v", got, ok)
	}
	registry.Clear()
	if _, ok := registry.Get("button"); ok {
		t.Fatal("cleared registry returned an asset")
	}
}
