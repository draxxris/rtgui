package layout

import (
	"fmt"
	"math"

	"github.com/draxxris/rtgui/core"
)

// ConstraintTolerance is the absolute logical-pixel tolerance used when
// checking additional point equations against the solved origin and size.
const ConstraintTolerance float32 = 0.001

type layoutAxis uint8

const (
	horizontalAxis layoutAxis = iota
	verticalAxis
)

type axisEquation struct {
	factor float32
	value  float32
	point  Anchor
}

// Arrange resolves node relative to parentBounds, then resolves every owned
// descendant once in cached dependency order.
func Arrange(node *Node, parentBounds core.Rect) error {
	return arrangeTree(node, parentBounds, false)
}

// ArrangeRoot fixes the root to viewport and resolves every descendant once.
// Root point and size constraints do not override the supplied viewport.
func ArrangeRoot(node *Node, viewport core.Rect) error {
	return arrangeTree(node, viewport, true)
}

// arrangeTree validates the external rectangle, obtains dependency order, and
// performs one measure-and-resolve visit per node.
func arrangeTree(root *Node, external core.Rect, fixedRoot bool) error {
	if root == nil {
		return ErrNilNode
	}
	if err := validateExternalRect(external); err != nil {
		return err
	}
	order, err := dependencyOrder(root)
	if err != nil {
		return err
	}
	revision := root.revision
	for _, node := range order {
		resolved := external
		if node != root || !fixedRoot {
			resolved, err = resolveNode(node, root, external)
			if err != nil {
				return err
			}
		}
		changed := !node.arranged || node.resolved != resolved
		node.resolved = resolved
		node.arranged = true
		if changed && node.onResize != nil {
			node.onResize(node)
		}
		// Ownership callbacks can invalidate the cached order. Leave dirty for
		// the next arrangement instead of reading the invalidated slice.
		if root.revision != revision {
			return nil
		}
	}
	if root.revision == revision {
		root.dirty = false
	}
	return nil
}

// resolveNode solves the two independent axes against already-resolved
// ownership and point dependencies.
func resolveNode(node, root *Node, external core.Rect) (core.Rect, error) {
	parent := external
	if node != root {
		parent = node.parent.resolved
	}
	x, width, err := solveAxis(node, root, external, parent, horizontalAxis)
	if err != nil {
		return core.Rect{}, err
	}
	y, height, err := solveAxis(node, root, external, parent, verticalAxis)
	if err != nil {
		return core.Rect{}, err
	}
	return core.Rect{X: x, Y: y, W: width, H: height}, nil
}

// solveAxis gathers at most nine equations without allocation, then uses a
// preferred size or two independent source factors as required.
func solveAxis(node, root *Node, external, parent core.Rect, axis layoutAxis) (float32, float32, error) {
	var equations [9]axisEquation
	count := collectAxisEquations(node, root, external, axis, &equations)
	if count == 0 {
		size, err := preferredAxisSize(node, parent, axis)
		if err != nil {
			return 0, 0, err
		}
		origin := parentOrigin(parent, axis) + authoredOrigin(node, axis)
		if !finite(origin) {
			return 0, 0, fmt.Errorf("%w: node %q has non-finite %s origin", ErrInvalidSize, node.id, axisName(axis))
		}
		return origin, size, nil
	}
	independent := independentEquation(equations[:count])
	if independent < 0 {
		return solveAxisWithPreferred(node, parent, axis, equations[:count])
	}
	return solveAxisFromPoints(node, axis, equations[:count], independent)
}

// collectAxisEquations converts point relations into scalar equations using
// target bounds that dependency ordering has already resolved.
func collectAxisEquations(node, root *Node, external core.Rect, axis layoutAxis, out *[9]axisEquation) int {
	for index, relation := range node.points {
		targetBounds := external
		if relation.target != nil {
			targetBounds = relation.target.resolved
		} else if node != root {
			targetBounds = node.parent.resolved
		}
		sourceX, sourceY := anchorFactors(relation.source)
		targetX, targetY := anchorFactors(relation.targetPoint)
		if axis == horizontalAxis {
			out[index] = axisEquation{factor: sourceX, value: targetBounds.X + targetX*targetBounds.W + relation.offset.X, point: relation.source}
		} else {
			out[index] = axisEquation{factor: sourceY, value: targetBounds.Y + targetY*targetBounds.H + relation.offset.Y, point: relation.source}
		}
	}
	return len(node.points)
}

func independentEquation(equations []axisEquation) int {
	for index := 1; index < len(equations); index++ {
		if equations[index].factor != equations[0].factor {
			return index
		}
	}
	return -1
}

// solveAxisWithPreferred positions a fixed, relative, or authored size and
// verifies every same-factor equation against the documented tolerance.
func solveAxisWithPreferred(node *Node, parent core.Rect, axis layoutAxis, equations []axisEquation) (float32, float32, error) {
	size, err := preferredAxisSize(node, parent, axis)
	if err != nil {
		return 0, 0, err
	}
	origin := equations[0].value - equations[0].factor*size
	if err := validateEquations(node, axis, equations, origin, size); err != nil {
		return 0, 0, err
	}
	return origin, size, nil
}

// solveAxisFromPoints derives origin and size from the first two independent
// factors, validates limits, and checks all additional equations.
func solveAxisFromPoints(node *Node, axis layoutAxis, equations []axisEquation, independent int) (float32, float32, error) {
	first, second := equations[0], equations[independent]
	size := (second.value - first.value) / (second.factor - first.factor)
	if !finite(size) {
		return 0, 0, fmt.Errorf("%w: node %q has non-finite %s", ErrInvalidSize, node.id, axisName(axis))
	}
	if size < 0 {
		return 0, 0, fmt.Errorf("%w: node %q %s=%g", ErrNegativeSize, node.id, axisName(axis), size)
	}
	if err := validatePointSize(node, axis, size); err != nil {
		return 0, 0, err
	}
	origin := first.value - first.factor*size
	if err := validateEquations(node, axis, equations, origin, size); err != nil {
		return 0, 0, err
	}
	return origin, size, nil
}

// validateEquations checks all point relations against one scalar solution.
func validateEquations(node *Node, axis layoutAxis, equations []axisEquation, origin, size float32) error {
	if !finite(origin) || !finite(size) {
		return fmt.Errorf("%w: node %q has non-finite %s solution", ErrInvalidSize, node.id, axisName(axis))
	}
	for _, equation := range equations {
		actual := origin + equation.factor*size
		if !finite(equation.value) || !finite(actual) {
			return fmt.Errorf("%w: node %q has non-finite %s equation", ErrConstraintConflict, node.id, axisName(axis))
		}
		if float32(math.Abs(float64(actual-equation.value))) > ConstraintTolerance {
			return fmt.Errorf("%w: node %q %s point %s differs by %g", ErrConstraintConflict, node.id, axisName(axis), AnchorName(equation.point), actual-equation.value)
		}
	}
	return nil
}

// preferredAxisSize applies fixed-over-relative-over-authored precedence and
// clamps that under-constrained size to valid minimum and maximum limits.
func preferredAxisSize(node *Node, parent core.Rect, axis layoutAxis) (float32, error) {
	size := authoredSize(node, axis)
	if node.hasRelative {
		size = parentSize(parent, axis) * vectorAxis(node.relativeSize, axis)
	}
	if node.hasFixedSize {
		size = vectorAxis(node.fixedSize, axis)
	}
	minimum, maximum, hasMinimum, hasMaximum, err := axisLimits(node, axis)
	if err != nil {
		return 0, err
	}
	if !finite(size) {
		return 0, fmt.Errorf("%w: node %q has non-finite %s", ErrInvalidSize, node.id, axisName(axis))
	}
	if size < 0 {
		return 0, fmt.Errorf("%w: node %q %s=%g", ErrNegativeSize, node.id, axisName(axis), size)
	}
	if hasMinimum && size < minimum {
		size = minimum
	}
	if hasMaximum && size > maximum {
		size = maximum
	}
	return size, nil
}

// validatePointSize rejects, rather than clamps, a size determined by two
// independent point equations.
func validatePointSize(node *Node, axis layoutAxis, size float32) error {
	minimum, maximum, hasMinimum, hasMaximum, err := axisLimits(node, axis)
	if err != nil {
		return err
	}
	if hasMinimum && size < minimum {
		return fmt.Errorf("%w: node %q %s=%g minimum=%g", ErrMinimumSize, node.id, axisName(axis), size, minimum)
	}
	if hasMaximum && size > maximum {
		return fmt.Errorf("%w: node %q %s=%g maximum=%g", ErrMaximumSize, node.id, axisName(axis), size, maximum)
	}
	return nil
}

// axisLimits validates and returns one axis of the optional size limits.
func axisLimits(node *Node, axis layoutAxis) (float32, float32, bool, bool, error) {
	minimum := vectorAxis(node.minSize, axis)
	maximum := vectorAxis(node.maxSize, axis)
	if node.hasMinSize && (!finite(minimum) || minimum < 0) {
		return 0, 0, false, false, fmt.Errorf("%w: node %q has invalid minimum %s", ErrInvalidSize, node.id, axisName(axis))
	}
	if node.hasMaxSize && (!finite(maximum) || maximum < 0) {
		return 0, 0, false, false, fmt.Errorf("%w: node %q has invalid maximum %s", ErrInvalidSize, node.id, axisName(axis))
	}
	if node.hasMinSize && node.hasMaxSize && minimum > maximum {
		return 0, 0, false, false, fmt.Errorf("%w: node %q minimum exceeds maximum on %s", ErrInvalidSize, node.id, axisName(axis))
	}
	return minimum, maximum, node.hasMinSize, node.hasMaxSize, nil
}

// validateExternalRect rejects unusable parent or viewport geometry.
func validateExternalRect(rect core.Rect) error {
	if !finite(rect.X) || !finite(rect.Y) || !finite(rect.W) || !finite(rect.H) {
		return fmt.Errorf("%w: non-finite parent bounds", ErrInvalidSize)
	}
	if rect.W < 0 || rect.H < 0 {
		return fmt.Errorf("%w: parent bounds %gx%g", ErrNegativeSize, rect.W, rect.H)
	}
	return nil
}

func vectorAxis(vector core.Vec2, axis layoutAxis) float32 {
	if axis == horizontalAxis {
		return vector.X
	}
	return vector.Y
}

func authoredOrigin(node *Node, axis layoutAxis) float32 {
	if axis == horizontalAxis {
		return node.authored.X
	}
	return node.authored.Y
}

func authoredSize(node *Node, axis layoutAxis) float32 {
	if axis == horizontalAxis {
		return node.authored.W
	}
	return node.authored.H
}

func parentOrigin(parent core.Rect, axis layoutAxis) float32 {
	if axis == horizontalAxis {
		return parent.X
	}
	return parent.Y
}

func parentSize(parent core.Rect, axis layoutAxis) float32 {
	if axis == horizontalAxis {
		return parent.W
	}
	return parent.H
}

func axisName(axis layoutAxis) string {
	if axis == horizontalAxis {
		return "width"
	}
	return "height"
}

// MoveFrame authors a translation by shifting every active point offset, or
// the single authored position when the node has no points. Arrangement is
// required before resolved bounds change.
func MoveFrame(node *Node, delta core.Vec2) {
	if node == nil || delta == (core.Vec2{}) {
		return
	}
	if len(node.points) == 0 {
		node.authored.X += delta.X
		node.authored.Y += delta.Y
	} else {
		for index := range node.points {
			node.points[index].offset.X += delta.X
			node.points[index].offset.Y += delta.Y
		}
	}
	node.markDirty(false)
}
