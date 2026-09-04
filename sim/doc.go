// Package sim provides a headless harness that simulates human input
// against widgets.
//
// A Stage is an instance-owned registry that maps a string name to a
// widget and drives the existing widget primitives (hover, press,
// release, focus, per-rune typing) plus an instance-owned input capture.
// There is no package-global state: each Stage owns its map and its
// capture, so tests can run in parallel with independent stages.
//
// Clicking uses the widget center (center-click v1). Future extensions
// may add explicit-position clicks for small, transformed, or clipped
// widgets; the current behavior is documented so callers do not depend
// on clipping or transform handling.
//
// Typing uses focus-first append semantics: the target textbox is
// focused (blurring the previously focused widget) and each rune of the
// input is appended via TypeChar, which keeps multi-byte UTF-8 safe.
//
// Concurrency: Stage protects its registry map, insertion order, and
// focused pointer with an RWMutex. The borrowed or owned input.Capture
// has its own internal mutex. Widget values themselves have no internal
// locking; concurrent use of the same widget through a Stage and through
// direct caller references is the caller's responsibility. Holding the
// Stage lock does not make widget mutation thread-safe by itself.
package sim
