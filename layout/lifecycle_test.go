package layout

import (
	"github.com/draxxris/rtgui/core"
	"testing"
)

// TestResizeMutationStopsInvalidatedTraversal guards callbacks that remove nodes.
func TestResizeMutationStopsInvalidatedTraversal(t *testing.T) {
	root := New("root", core.Rect{W: 100, H: 100})
	child := New("child", core.Rect{W: 20, H: 20})
	if err := root.AddChild(child); err != nil {
		t.Fatal(err)
	}
	root.SetOnResize(func(*Node) { root.RemoveChild(child) })
	if err := Arrange(root, core.Rect{}); err != nil {
		t.Fatal(err)
	}
	if child.Parent() != nil {
		t.Fatal("callback failed to detach child")
	}
	if err := Arrange(root, core.Rect{}); err != nil {
		t.Fatal(err)
	}
}

// TestResizeFiresOnlyOnBoundsChange avoids redundant layout callback work.
func TestResizeFiresOnlyOnBoundsChange(t *testing.T) {
	root := New("root", core.Rect{W: 100, H: 100})
	calls := 0
	root.SetOnResize(func(*Node) { calls++ })
	for i := 0; i < 3; i++ {
		if err := Arrange(root, core.Rect{}); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("unchanged layout fired %d callbacks", calls)
	}
}
