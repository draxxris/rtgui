package layout

import (
	"errors"
	"fmt"
	"math"

	"github.com/draxxris/rtgui/core"
)

// Anchor identifies one of the nine normalized points on a layout node.
type Anchor int

const (
	// AnchorTopLeft is the top-left point.
	AnchorTopLeft Anchor = iota
	// AnchorTop is the midpoint of the top edge.
	AnchorTop
	// AnchorTopRight is the top-right point.
	AnchorTopRight
	// AnchorLeft is the midpoint of the left edge.
	AnchorLeft
	// AnchorCenter is the center point.
	AnchorCenter
	// AnchorRight is the midpoint of the right edge.
	AnchorRight
	// AnchorBottomLeft is the bottom-left point.
	AnchorBottomLeft
	// AnchorBottom is the midpoint of the bottom edge.
	AnchorBottom
	// AnchorBottomRight is the bottom-right point.
	AnchorBottomRight
)

var (
	// ErrNilNode reports an operation that requires a layout node.
	ErrNilNode = errors.New("layout: nil node")
	// ErrNilChild reports an attempt to add a nil ownership child.
	ErrNilChild = errors.New("layout: nil child")
	// ErrMultipleParents reports an attempt to give a node two ownership parents.
	ErrMultipleParents = errors.New("layout: child already has a parent")
	// ErrOwnershipCycle reports an attempted or discovered ownership cycle.
	ErrOwnershipCycle = errors.New("layout: ownership cycle")
	// ErrInvalidAnchor reports a point outside the Anchor enum.
	ErrInvalidAnchor = errors.New("layout: invalid anchor")
	// ErrAnchorTargetOutsideTree reports a relation to a node outside the arranged tree.
	ErrAnchorTargetOutsideTree = errors.New("layout: anchor target outside arranged tree")
	// ErrAnchorCycle reports a cycle in point and ownership dependencies.
	ErrAnchorCycle = errors.New("layout: anchor dependency cycle")
	// ErrConstraintConflict reports point equations that cannot all be satisfied.
	ErrConstraintConflict = errors.New("layout: conflicting point constraints")
	// ErrNegativeSize reports a preferred or point-derived negative size.
	ErrNegativeSize = errors.New("layout: negative size")
	// ErrMinimumSize reports a point-derived size below its minimum.
	ErrMinimumSize = errors.New("layout: point-derived size below minimum")
	// ErrMaximumSize reports a point-derived size above its maximum.
	ErrMaximumSize = errors.New("layout: point-derived size above maximum")
	// ErrInvalidSize reports non-finite or contradictory size configuration.
	ErrInvalidSize = errors.New("layout: invalid size configuration")
)

type pointRelation struct {
	source      Anchor
	target      *Node
	targetPoint Anchor
	offset      core.Vec2
}

type dependencyFrame struct {
	node *Node
	next int
}

type treeCache struct {
	graphDirty bool
	generation uint64
	nodes      []*Node
	order      []*Node
	scan       []*Node
	stack      []dependencyFrame
}

// Node is one authored frame in an ownership tree. Constraint and ownership
// state is private so every mutation can invalidate the appropriate root cache.
type Node struct {
	id       string
	authored core.Rect
	resolved core.Rect
	arranged bool

	fixedSize    core.Vec2
	hasFixedSize bool
	relativeSize core.Vec2
	hasRelative  bool
	minSize      core.Vec2
	hasMinSize   bool
	maxSize      core.Vec2
	hasMaxSize   bool

	points   []pointRelation
	children []*Node
	parent   *Node
	onResize func(*Node)

	dirty    bool
	revision uint64
	cache    treeCache

	membershipRoot       *Node
	membershipGeneration uint64
	visitRoot            *Node
	visitGeneration      uint64
	visitState           uint8
}

// New returns a detached layout node with authored bounds.
func New(id string, bounds core.Rect) *Node {
	node := &Node{id: id, authored: bounds, dirty: true}
	node.cache.graphDirty = true
	return node
}

// ID returns the node's diagnostic identity.
func (n *Node) ID() string {
	if n == nil {
		return ""
	}
	return n.id
}

// Bounds returns resolved bounds after arrangement and authored bounds before it.
func (n *Node) Bounds() core.Rect {
	if n == nil {
		return core.Rect{}
	}
	if n.arranged {
		return n.resolved
	}
	return n.authored
}

// AuthoredBounds returns the node's local position and preferred size.
func (n *Node) AuthoredBounds() core.Rect {
	if n == nil {
		return core.Rect{}
	}
	return n.authored
}

// ResolvedBounds returns the bounds written by the latest successful arrangement.
func (n *Node) ResolvedBounds() core.Rect {
	if n == nil {
		return core.Rect{}
	}
	return n.resolved
}

// SetBounds replaces the single authored position and size representation.
func (n *Node) SetBounds(bounds core.Rect) {
	if n == nil || n.authored == bounds {
		return
	}
	n.authored = bounds
	n.markDirty(false)
}

// Parent returns the ownership parent, or nil for an ownership root.
func (n *Node) Parent() *Node {
	if n == nil {
		return nil
	}
	return n.parent
}

// Children returns a snapshot of the ownership children in registration order.
func (n *Node) Children() []*Node {
	if n == nil {
		return nil
	}
	return append([]*Node(nil), n.children...)
}

// AddChild appends one detached child and rejects nil children, multiple
// parents, and ownership cycles before changing either tree.
func (n *Node) AddChild(child *Node) error {
	if n == nil {
		return ErrNilNode
	}
	if child == nil {
		return ErrNilChild
	}
	if child.parent != nil {
		return fmt.Errorf("%w: %q", ErrMultipleParents, child.id)
	}
	for ancestor := n; ancestor != nil; ancestor = ancestor.parent {
		if ancestor == child {
			return fmt.Errorf("%w: adding %q under %q", ErrOwnershipCycle, child.id, n.id)
		}
	}
	child.parent = n
	n.children = append(n.children, child)
	child.dirty = true
	child.cache.graphDirty = true
	n.markDirty(true)
	return nil
}

// RemoveChild detaches a direct child and invalidates both resulting trees.
func (n *Node) RemoveChild(child *Node) bool {
	if n == nil || child == nil || child.parent != n {
		return false
	}
	for index, candidate := range n.children {
		if candidate != child {
			continue
		}
		copy(n.children[index:], n.children[index+1:])
		n.children = n.children[:len(n.children)-1]
		child.parent = nil
		n.markDirty(true)
		child.markDirty(true)
		return true
	}
	return false
}

// SetPoint relates one source point to a target point plus offset. A nil target
// resolves against the ownership parent. Replacing a source keeps its order.
func (n *Node) SetPoint(source Anchor, target *Node, targetPoint Anchor, offset core.Vec2) error {
	if n == nil {
		return ErrNilNode
	}
	if !validAnchor(source) || !validAnchor(targetPoint) {
		return fmt.Errorf("%w: source=%d target=%d", ErrInvalidAnchor, source, targetPoint)
	}
	if target == n {
		return fmt.Errorf("%w: node %q targets itself", ErrAnchorCycle, n.id)
	}
	if !finite(offset.X) || !finite(offset.Y) {
		return fmt.Errorf("%w: non-finite point offset on %q", ErrConstraintConflict, n.id)
	}
	relation := pointRelation{source: source, target: target, targetPoint: targetPoint, offset: offset}
	for index := range n.points {
		if n.points[index].source != source {
			continue
		}
		if n.points[index] == relation {
			return nil
		}
		n.points[index] = relation
		n.markDirty(true)
		return nil
	}
	n.points = append(n.points, relation)
	n.markDirty(true)
	return nil
}

// ClearPoint removes the relation for source while retaining relation capacity.
func (n *Node) ClearPoint(source Anchor) error {
	if n == nil {
		return ErrNilNode
	}
	if !validAnchor(source) {
		return fmt.Errorf("%w: source=%d", ErrInvalidAnchor, source)
	}
	for index := range n.points {
		if n.points[index].source != source {
			continue
		}
		copy(n.points[index:], n.points[index+1:])
		n.points = n.points[:len(n.points)-1]
		n.markDirty(true)
		return nil
	}
	return nil
}

// ClearAllPoints removes every active point relation while retaining capacity.
func (n *Node) ClearAllPoints() {
	if n == nil || len(n.points) == 0 {
		return
	}
	n.points = n.points[:0]
	n.markDirty(true)
}

// SetAllPoints replaces all relations with zero-offset top-left and
// bottom-right relations to target. A nil target means the ownership parent.
func (n *Node) SetAllPoints(target *Node) error {
	if n == nil {
		return ErrNilNode
	}
	if target == n {
		return fmt.Errorf("%w: node %q targets itself", ErrAnchorCycle, n.id)
	}
	n.points = n.points[:0]
	n.points = append(n.points,
		pointRelation{source: AnchorTopLeft, target: target, targetPoint: AnchorTopLeft},
		pointRelation{source: AnchorBottomRight, target: target, targetPoint: AnchorBottomRight},
	)
	n.markDirty(true)
	return nil
}

// SetFixedSize selects an absolute preferred size for under-constrained axes.
func (n *Node) SetFixedSize(size core.Vec2) {
	if n == nil || (n.hasFixedSize && n.fixedSize == size) {
		return
	}
	n.fixedSize, n.hasFixedSize = size, true
	n.markDirty(false)
}

// ClearFixedSize restores relative or authored preferred sizing.
func (n *Node) ClearFixedSize() {
	if n == nil || !n.hasFixedSize {
		return
	}
	n.hasFixedSize = false
	n.markDirty(false)
}

// SetRelativeSize selects parent-relative preferred sizing for under-constrained axes.
func (n *Node) SetRelativeSize(size core.Vec2) {
	if n == nil || (n.hasRelative && n.relativeSize == size) {
		return
	}
	n.relativeSize, n.hasRelative = size, true
	n.markDirty(false)
}

// ClearRelativeSize restores authored preferred sizing when no fixed size exists.
func (n *Node) ClearRelativeSize() {
	if n == nil || !n.hasRelative {
		return
	}
	n.hasRelative = false
	n.markDirty(false)
}

// SetMinSize clamps preferred sizes and validates point-derived sizes.
func (n *Node) SetMinSize(size core.Vec2) {
	if n == nil || (n.hasMinSize && n.minSize == size) {
		return
	}
	n.minSize, n.hasMinSize = size, true
	n.markDirty(false)
}

// ClearMinSize removes the minimum-size limit.
func (n *Node) ClearMinSize() {
	if n == nil || !n.hasMinSize {
		return
	}
	n.hasMinSize = false
	n.markDirty(false)
}

// SetMaxSize clamps preferred sizes and validates point-derived sizes.
func (n *Node) SetMaxSize(size core.Vec2) {
	if n == nil || (n.hasMaxSize && n.maxSize == size) {
		return
	}
	n.maxSize, n.hasMaxSize = size, true
	n.markDirty(false)
}

// ClearMaxSize removes the maximum-size limit.
func (n *Node) ClearMaxSize() {
	if n == nil || !n.hasMaxSize {
		return
	}
	n.hasMaxSize = false
	n.markDirty(false)
}

// SetOnResize replaces the synchronous callback run after a node is resolved.
func (n *Node) SetOnResize(callback func(*Node)) {
	if n != nil {
		n.onResize = callback
	}
}

// PointCount reports the number of active source-point relations.
func (n *Node) PointCount() int {
	if n == nil {
		return 0
	}
	return len(n.points)
}

// NeedsArrange reports whether a mutation has made the ownership root stale.
func (n *Node) NeedsArrange() bool {
	if n == nil {
		return false
	}
	return n.ownershipRoot().dirty
}

// markDirty invalidates layout and, when requested, the cached dependency order.
func (n *Node) markDirty(graph bool) {
	if n == nil {
		return
	}
	root := n.ownershipRoot()
	root.dirty = true
	root.revision++
	if graph {
		root.cache.graphDirty = true
	}
}

func (n *Node) ownershipRoot() *Node {
	for n.parent != nil {
		n = n.parent
	}
	return n
}

func validAnchor(anchor Anchor) bool {
	return anchor >= AnchorTopLeft && anchor <= AnchorBottomRight
}

func finite(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}
