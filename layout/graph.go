package layout

import "fmt"

const (
	visitUnseen uint8 = iota
	visitActive
	visitDone
)

// dependencyOrder returns a cached topological order and rebuilds it only
// after ownership or point-relation mutations.
func dependencyOrder(root *Node) ([]*Node, error) {
	cache := &root.cache
	if !cache.graphDirty && len(cache.order) != 0 {
		return cache.order, nil
	}
	cache.generation++
	cache.nodes = cache.nodes[:0]
	cache.order = cache.order[:0]
	if err := collectOwnership(root, cache); err != nil {
		return nil, err
	}
	if err := validateTargets(root, cache); err != nil {
		return nil, err
	}
	if err := sortDependencies(root, cache); err != nil {
		return nil, err
	}
	cache.graphDirty = false
	return cache.order, nil
}

// collectOwnership scans the tree in registration order while checking the
// private parent/child invariant defensively.
func collectOwnership(root *Node, cache *treeCache) error {
	cache.scan = append(cache.scan[:0], root)
	for len(cache.scan) != 0 {
		last := len(cache.scan) - 1
		node := cache.scan[last]
		cache.scan = cache.scan[:last]
		if node == nil {
			return ErrNilChild
		}
		if node.membershipRoot == root && node.membershipGeneration == cache.generation {
			return fmt.Errorf("%w at %q", ErrOwnershipCycle, node.id)
		}
		node.membershipRoot = root
		node.membershipGeneration = cache.generation
		cache.nodes = append(cache.nodes, node)
		for index := len(node.children) - 1; index >= 0; index-- {
			child := node.children[index]
			if child == nil {
				return fmt.Errorf("%w under %q", ErrNilChild, node.id)
			}
			if child.parent != node {
				return fmt.Errorf("%w: child %q is not owned by %q", ErrMultipleParents, child.id, node.id)
			}
			cache.scan = append(cache.scan, child)
		}
	}
	return nil
}

// validateTargets rejects every explicit relation whose target is absent from
// the ownership tree being arranged.
func validateTargets(root *Node, cache *treeCache) error {
	for _, node := range cache.nodes {
		for _, relation := range node.points {
			target := relation.target
			if target == nil && node != root {
				target = node.parent
			}
			if target == nil {
				continue
			}
			if target.membershipRoot != root || target.membershipGeneration != cache.generation {
				return fmt.Errorf("%w: node %q %s targets %q", ErrAnchorTargetOutsideTree, node.id, AnchorName(relation.source), target.id)
			}
		}
	}
	return nil
}

// sortDependencies performs an iterative depth-first topological sort. The
// stack and per-node marks are reused on later graph rebuilds.
func sortDependencies(root *Node, cache *treeCache) error {
	for _, start := range cache.nodes {
		if nodeVisitState(start, root, cache.generation) == visitDone {
			continue
		}
		setNodeVisitState(start, root, cache.generation, visitActive)
		cache.stack = append(cache.stack[:0], dependencyFrame{node: start})
		for len(cache.stack) != 0 {
			frame := &cache.stack[len(cache.stack)-1]
			dependency, more := nextDependency(frame, root)
			if more {
				if dependency == nil {
					continue
				}
				state := nodeVisitState(dependency, root, cache.generation)
				if state == visitActive {
					return fmt.Errorf("%w: %q depends on %q", ErrAnchorCycle, frame.node.id, dependency.id)
				}
				if state == visitUnseen {
					setNodeVisitState(dependency, root, cache.generation, visitActive)
					cache.stack = append(cache.stack, dependencyFrame{node: dependency})
				}
				continue
			}
			setNodeVisitState(frame.node, root, cache.generation, visitDone)
			cache.order = append(cache.order, frame.node)
			cache.stack = cache.stack[:len(cache.stack)-1]
		}
	}
	return nil
}

// nextDependency emits the ownership parent first, followed by point targets
// in stable relation order. Nil targets on the arranged root are external.
func nextDependency(frame *dependencyFrame, root *Node) (*Node, bool) {
	node := frame.node
	if node != root {
		if frame.next == 0 {
			frame.next++
			return node.parent, true
		}
		relationIndex := frame.next - 1
		if relationIndex >= len(node.points) {
			return nil, false
		}
		frame.next++
		target := node.points[relationIndex].target
		if target == nil {
			target = node.parent
		}
		return target, true
	}
	if frame.next >= len(node.points) {
		return nil, false
	}
	target := node.points[frame.next].target
	frame.next++
	return target, true
}

func nodeVisitState(node, root *Node, generation uint64) uint8 {
	if node.visitRoot != root || node.visitGeneration != generation {
		return visitUnseen
	}
	return node.visitState
}

func setNodeVisitState(node, root *Node, generation uint64, state uint8) {
	node.visitRoot = root
	node.visitGeneration = generation
	node.visitState = state
}
