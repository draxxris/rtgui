# Interaction and cache contracts

All operations use one owning goroutine. Apply network updates on that goroutine,
not from background packet handlers.

## Migration

- A widget can belong to only one UI.
- Widgets own one callback registry. `button.OnClick(fn)` and `u.OnClick(name, fn)`
  replace the same slot. They do not register two listeners.
- Register the widget before using a named callback setter. Unknown names have no callback storage.
- Removal clears callbacks, tooltip data, drag bindings, and scoped hotkeys.
  Re-register callbacks when reusing a removed widget.
- Global hotkeys remain until `RemoveHotkey` or destruction of the UI.
- A frame owns children through layout ancestry, not their screen positions.
- `Remove` disposes registered descendants. `SetVisible(false)` hides a subtree without destroying it.
- Disabled overlays still block underlying input. Decorative overlays can explicitly enable input transparency.
- A captured gesture consumes its release, including cancellation after a widget becomes disabled.
- Editing keys belong to the focused textbox even when no mutation is possible.
  Mutation callbacks remain silent for rejected edits.
- Scroll limits always clamp offsets. Set a positive maximum before scrolling.
- Standalone drag targets use last-registration precedence, matching drawing order.

## Parenting and clipping

Use `UI.SetParent` for registered widgets. The same relation is available through
`parent.Frame().AddChild(child.Frame())` before registration. Do not link nodes across UIs.
Author child bounds in parent-local coordinates, then call `layout.Arrange` on the root.

Drawing and pointer hit testing use the same tree and sibling order. A child cannot
draw or receive pointer input outside its parent's clip. Scroll panels exclude their
scrollbar from the child clip and translate descendants by their scroll offset.

`BringToFront` raises a complete root subtree. Child registration order cannot move
a background window's child above a foreground window.

The host applies the logical drawing matrix, as shown in the gallery. Scissor conversion
includes viewport scale and translation. `Theme.PushClip` and `PopClip` preserve nested clips.
Do not mix this stack with unmanaged raylib scissor calls inside content drawers.

## Drag lifetime

Register sources with `OnDrag` and targets with `OnDrop`. The UI owns gesture routing.
Do not separately feed the same gesture into the controller.

A source factory runs on press. Keep it small and free of UI mutations.
Crossing the threshold suppresses the original click. Dropping closes the old session
before invoking delivery, so a callback cannot overwrite a newer session.

Acceptance predicates provide previews and final validation. They must be pure.
A foreground rejection does not search behind that surface. Targets can bubble to
ownership ancestors, which permits dropping on an empty region of a bag.

The application supplies ghost drawing through `SetDragGhostDrawer`. It can query
`CurrentTarget` and `CanDrop` to show valid or invalid feedback. Ghosts do not block input.

The controller releases payload references on drop or cancellation. Delivery reports intent only.
Inventory changes, server rejection, pending state, and rollback remain application responsibilities.

`CancelInput` releases capture on OS focus loss. `PollRaylibInput` calls it automatically.
Manual input adapters must call it when their input device loses focus.

## Cache and memory lifetime

- Text buffers build an immutable string only after text changes. Earlier strings remain valid.
- Rich-text drawing and hit testing share a per-widget revisioned layout.
  Text, bounds, skin insets, font changes, and graphics readiness invalidate it.
  Inline icon and link tints resolve at draw time from parent registries, so
  re-registering recolors without re-parsing segments.
- Player markup parses only `[icon=name]` and `[link=scheme:target]text[/link]`
  via `text.ParsePlayerMarkup`; color, bold, size, and face have no markup
  and are Go-constructed `RichSegment` fields. Unknown tags render literally.
- Every widget carries optional rich runs; buttons, labels, checkboxes, and
  dropdown rows render single-line rich runs. Clickable links activate only
  in `RichText` and the read-only `ChatLog`; other controls render link
  colors without underlines.
- Chat logs keep bounded message history with stable IDs until eviction.
  The UI reconciles scroll limits from theme-measured variable heights in
  two passes (full content, then track-excluded rows) so wrapping, drawing,
  and hit testing share one viewport. Per-message heights and segment copies
  reuse UI scratch storage; eviction zeroes truncated tails. Message tooltips
  and press arms track message IDs, not indices, so eviction never
  mis-highlights.
- Public snapshot getters still copy. Internal tab and dropdown drawing reuse scratch buffers.
- Formatted slider and progress labels update only when their value or format changes.
- Rich tooltips copy supplied segments. Cursor movement translates cached geometry without wrapping again.
- Graphs copy input data and cache mapped points and tick labels.
  Fixed-range streaming can reuse these buffers after warmup.
- Graph history is bounded by default. Bounded appends currently shift retained points,
  so append cost grows with the configured history size.
- Diagnostic history retains at most 128 unique recent messages. Missing-skin reporting
  retains at most 128 distinct keys per theme and suppresses repeats.
- Widget removal releases UI cache entries. Retained capacity in reusable buffers is intentional.
- Font atlases have a bounded size cache. Graphics resources still require explicit unloading.

### Table view and column-cache lifetime

Tables retain insertion-order rows and build a separate stable-ID sort view; sorting
never mutates the game's row order. Numeric columns sort by a cell's
`SortValue` only when `HasSortValue` is true. Missing numeric values go last and
use their text as a deterministic tie-breaker, so formatted currency must not be
parsed from display text.

The UI owns the one resolved-column cache for the lifetime of the registered table/UI
pair. It is invalidated when columns, bounds, scrollbar presence, skin
padding/state/class, or pixel snapping changes, and the entry is released when
the widget is removed. The UI passes those track-excluded slots to the typed
renderer draw; a direct renderer call without slots resolves a one-shot fallback.
Draw and hit testing therefore use the same slots, and warmed virtual draw/hover
paths reuse the UI-owned storage.

Table pointer input follows the armed press-release (MSFT) rule: a press arms a
row or header identity, and only a release on that same identity commits. A
mismatched release is consumed without selecting or sorting. Row release only
selects; activation is a game-owned semantic call such as
`ActivateTableRow`. Tables have no double-click activation and no keyboard
navigation or keyboard activation in this version. Table scroll bounds are always
derived from `EnsureScrollBounds`/layout reconciliation; there is no explicit
Table max-scroll override. Sort directions use `core.SortAsc`, `core.SortDesc`,
and `core.None`.

Headless allocation tests cover warmed full-widget drawing, rich hover, tooltips, and graphs.
They do not measure native raylib allocations, font uploads, or GPU costs.

## Deferred features

The gallery includes only a scripted auction-table demonstration; this change does not add
production auction transactions, item grids, inventory transactions, or network logic.
Rich tooltips are non-interactive. Text shaping, IME support, and broader glyph coverage remain separate work.

The [roadmap](roadmap.md) records completed foundations and the remaining widget, renderer, and game-integration work.
