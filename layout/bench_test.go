package layout

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

var benchmarkBounds core.Rect

// BenchmarkArrangeWarmTree measures unchanged layout after ownership and
// dependency slices and traversal marks have been built once.
func BenchmarkArrangeWarmTree(b *testing.B) {
	root := New("root", core.Rect{W: 1280, H: 720})
	previous := root
	for index := 0; index < 32; index++ {
		node := New("node", core.Rect{W: 120, H: 32})
		if err := root.AddChild(node); err != nil {
			b.Fatal(err)
		}
		if err := node.SetPoint(AnchorTopLeft, previous, AnchorTopLeft, core.Vec2{X: 3, Y: 2}); err != nil {
			b.Fatal(err)
		}
		previous = node
	}
	viewport := root.AuthoredBounds()
	if err := ArrangeRoot(root, viewport); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if err := ArrangeRoot(root, viewport); err != nil {
			b.Fatal(err)
		}
	}
	benchmarkBounds = previous.Bounds()
}
