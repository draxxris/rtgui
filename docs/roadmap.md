# rtgui Roadmap — MMORPG UI

This roadmap records implemented capabilities and remaining work for an MMORPG client.
It covers widgets, interaction, rendering, memory use, and game integration.

- **DONE** means the stated capability exists in the current code.
- **PLANNED** means the capability remains unimplemented. Priority does not imply an
  implementation commitment.
- A completed foundation does not mark its dependent game feature as complete.

Runtime contracts and breaking changes are in [interaction.md](interaction.md).
Validation procedures are in [testing.md](testing.md).

## Design Principles

- Compose game panels from reusable widgets rather than adding a separate widget type
  for every screen.
- Keep widget callbacks in one registry. Named UI setters and fluent widget setters
  replace the same slots.
- Keep hover, capture, focus, and popup state in `ui.UI` on one owning goroutine.
- Use layout ancestry for parenting, clipping, inherited availability, and subtree
  stacking.
- Use UI-owned drag gestures for inventory and action-bar moves. Keep transaction rules
  in the game.
- Virtualize long lists and grids. Cache text layout, graph geometry, and formatted
  labels.
- Prefer reusable storage over object pools. Add pools only after allocation
  measurements justify them.
- Keep skins consistent across drawing, layout, and hit testing.
- Measure Go allocations and native graphics costs separately. Headless tests do not
  establish GPU performance.

## DONE — Core Widgets and Rendering

- **DONE — Basic controls**

  - `Button`, `Label`, `Checkbox`, `Textbox`, `Dropdown`, `Slider`, and `ProgressBar`.
  - `Frame`, `ScrollPanel`, and `Canvas` for containers and custom drawing.
  - Explicit text color, font size, italic style, and alignment on supported controls.

- **DONE — TabBar foundation**

  - Equal-width tabs, selection, per-cell visual states, and selection callbacks.
  - Overflow, closable tabs, badges, and automatic page management remain planned.

- **DONE — Context-menu foundation**

  - UI-owned flat menus, separators, disabled rows, and pointer selection.
  - Modal input routing and Escape dismissal.
  - Submenus, check items, and keyboard navigation remain planned.

- **DONE — Text editing and rich-text messages**

  - Bounded UTF-8 textbox storage, caret navigation, selection, and clipboard
    operations.
  - Focused editors consume editing intent even when no mutation is possible.
  - Rich-text messages support colored segments, wrapping, item/player/URL links, and
    link tooltips.
  - Drawing and hit testing share cached rich-text layout.
  - A virtualized chat log, text shaping, and IME support remain planned.

- **DONE — Rich tooltips**

  - Colored titles, subtitles, colored body segments, and borrowed atlas icons.
  - Widget hover content, explicitly anchored content, and explicit dismissal.
  - Cached layout, cursor-relative placement, edge flipping, and viewport clamping.
  - Tooltip links remain descriptive rather than interactive.
  - Hover delay, equipment comparison, and specialized item presentations remain
    planned.

- **DONE — Versatile line graph**

  - Multiple series with copied data, configurable colors, thickness, and visibility.
  - Automatic or explicit ranges, linear or step interpolation, and grid divisions.
  - Cached geometry, tick labels, custom tick formatting, and nearest-point queries.
  - Clipped rendering and a shared skin-aware plot rectangle.
  - Empty, constant, singleton, and non-finite data handling.
  - Each series defaults to 1,024 retained points. Unbounded history requires explicit
    opt-in.
  - Samples use `float32`. Authoritative currency and timestamps remain game-model data.

- **DONE — Layout, skins, and diagnostics**

  - `SetPoint` constraints, cached dependency ordering, arrangement, and frame movement.
  - Resize callbacks run only after a bounds change. Callback mutations stop invalidated
    traversal.
  - CSS skins, texture atlases, nine-patch geometry, and transactional CSS texture
    loading.
  - Explicit borrowed/owned texture lifetimes and a bounded optional draw recorder.

## DONE — Ownership, Input, and Memory Fixes

- **DONE — Single callback ownership and disposal**

  - Fluent and named callback setters share one widget-owned registry.
  - A widget cannot register with two UIs.
  - Removal disposes registered descendants, callbacks, scoped hotkeys, drag bindings,
    and tooltip caches.
  - Callback re-entry cannot activate a replacement widget through an old tab-selection
    event.

- **DONE — Parental frame ownership**

  - Registered widgets can use `UI.SetParent` with the existing layout tree.
  - Children inherit visibility, enabled state, clipping, and root stacking order.
  - Container focus follows ancestry instead of geometric overlap.
  - `BringToFront` raises the complete root subtree.

- **DONE — Overlay blocking and gesture capture**

  - Opaque overlays block underlying clicks and wheel input, including disabled
    overlays.
  - Decorative surfaces can explicitly enable input transparency.
  - Captured releases cannot become world clicks after cancellation or removal.
  - Right-click routing supports context menus and drag cancellation.
  - `CancelInput` releases interaction on focus loss. Raylib polling invokes it
    automatically.

- **DONE — UI-integrated drag and drop**

  - Widget-bound sources and targets share UI ownership, clipping, and stacking rules.
  - A movement threshold separates clicks from drags. A drag suppresses the original
    click.
  - Acceptance preview uses `CanDrop`. Rejection never searches unrelated background
    windows.
  - Applications supply ghost drawing without separate pointer routing.
  - Escape, right-click, source removal, inherited unavailability, and focus loss cancel
    gestures.
  - Delivery closes the old session before callbacks, preventing new-session corruption.
  - Terminal sessions release payload references. Standalone targets use
    last-registration precedence.
  - Accepted drop intent is not a committed inventory transaction.

- **DONE — Scroll ownership and removal**

  - Owned children use the same scroll offset for drawing and hit testing.
  - Nested clips intersect and restore the enclosing clip.
  - Removing or clearing panels releases scrollbar hover and drag references.
  - Scroll setters clamp offsets on every axis. A zero maximum prevents scrolling.

- **DONE — Allocation and retention improvements**

  - Textboxes cache immutable strings until text changes.
  - Slider and progress readouts cache formatted text until values or formats change.
  - Public snapshots remain defensive. Internal tab and dropdown drawing reuse scratch
    storage.
  - Rich-text drawing and hover share revisioned layout rather than rebuilding fragments
    every frame.
  - Input polling reuses character and hotkey buffers.
  - Removed entries and layout traversal scratch storage release stale references.
  - Diagnostic history and missing-skin reporting are bounded and suppress repeated
    messages.
  - Each font retains at most 32 raster sizes, capped at 256 pixels.

- **DONE — Regression coverage and examples**

  - Ownership, clipping, capture, removal, callback re-entry, and drag lifecycle tests.
  - Headless allocation tests for warmed UI frames, rich hover, tooltips, and graphs.
  - Fixed-range graph streaming reuses buffers after warmup.
  - Gallery examples for graphs, rich tooltips, and item-drop intent.
  - Migration documentation reflects callback disposal and parental ownership.

## P0 — PLANNED: Inventory and Window Foundations

### Window and Dialog

Frame ownership, input blocking, clipping, and subtree raising are DONE. The full window
widget still needs:

- Draggable title bars, close and pin buttons, and resizable edges.
- Saved position and size, viewport constraints, and minimum dimensions.
- A general modal stack, focus restoration, and keyboard focus traversal.
- Confirmation dialogs and stack-split quantity entry.
- Clear cancellation behavior for pending actions.

These controls support bags, bank, equipment, map, vendor, settings, mail, and auction
panels.

### ItemSlot and Virtualized ItemGrid

The generic drag controller and rich tooltip renderer are DONE. Item widgets still need:

- Item icons, rarity borders, stack counts, durability, and equipment-slot restrictions.
- Bound, locked, pending, selected, and cooldown states.
- Grid layout, virtualization, multi-selection, and right-click use.
- Valid-move, merge, split, swap, full-container, and invalid-slot feedback.
- Drag auto-scroll near container edges.
- Reuse across backpack, bank, equipment, loot, trade, vendor, and crafting panels.

### Virtualized ListView and TableView

- Visible-row reuse, sortable columns, and stable row identity.
- Single and multiple selection, icons, alternating rows, and aligned numeric columns.
- Incremental data updates that preserve selection and scroll position.
- Shared use by auction, guild roster, friends, LFG, mail, quests, and combat logs.

### ChatBox and Combat Log

Single-message rich text and its shared layout cache are DONE. Player
markup (`[icon=name]`, `[link=scheme:target]text[/link]` via
`text.ParsePlayerMarkup`), whitelisted inline icons, parent-registered
per-kind link colors, and single-line rich runs for buttons, labels,
checkboxes, and dropdown rows are DONE; clickable links activate in `RichText`
and the read-only `ChatLog` below.
Go-authored bold, size, and face survive as `RichSegment` fields. The read-only
log foundation is DONE: `ChatLog` carries bounded history (default 200, silent
oldest-first eviction, no overflow signal), variable-height word-wrapped
messages with a 4px gap, shared scrollbar gestures with lists and scroll
panels, bottom stick with break-on-scroll-up, press-arm/release link
activation with plain `func(core.Link)` callbacks, and hover link tooltips.
The log still needs:

- Virtualized multi-thousand history, channel filters, and unread indicators.
- Scroll anchoring when earlier history loads (bottom stick on arrival is DONE).
- Multiline input, input history, whisper completion, and slash commands.
- Rich-text selection and copying; international text support through the
  text-system work below.
- The windowed gallery shows the log on the About tab, sharing the
  scroll/graph region through locked slot-table placement.

### Advanced Rich Tooltip Behavior

- Configurable hover delay and stable transitions between nearby targets.
- Side-by-side equipment comparison, including modifier-key activation.
- Structured stat rows, requirements, binding state, and comparison deltas.
- Reusable talent, quest, currency, and ability presentations.

## P1 — PLANNED: General Controls and Notifications

- **TabControl extensions:** overflow scrolling, closable tabs, badges, and owned page
  switching.
- **Context-menu extensions:** nested submenus, check items, and keyboard navigation.
- **NumericInput / SpinBox:** validated integer entry, limits, increments, and keyboard
  editing.
- **SegmentedSelector / RadioGroup:** mutually exclusive choices independent of tab
  pages.
- **Settings controls:** toggle switches, keybind capture, and reusable standalone
  scrollbars.
- **Toasts:** bounded notification queues, timed dismissal, and optional actions. Reuse
  entries before considering pools.

## P1 — PLANNED: Architecture and Text Follow-up

### Widget Extensibility

Callback ownership and the concrete nil-check switch are fixed. Drawing and activation
still dispatch centrally by widget kind or concrete type.

- Decide whether the supported widget set is closed or externally extensible.
- For an extensible set, add small capabilities or registered handlers for custom
  drawing and interaction.
- Avoid a large plugin framework or another independent ownership tree.

### Renderer Boundary and Scaling

Nested scissor restoration and viewport-aware clip conversion are DONE. The UI still
depends on a concrete raylib theme, and the host applies the drawing matrix.

- Define one explicit contract for drawing transforms, DPI scaling, pixel snapping, and
  input mapping.
- Cover translated viewports and nonuniform scaling with framebuffer tests.
- Evaluate a smaller renderer boundary before supporting another 3D engine.
- Keep the current raylib path until a concrete integration requires replacement.

### International Text and Renderer Evaluation

Current font atlases cover printable ASCII and a small punctuation set. UTF-8 storage
does not imply complete glyph coverage or text shaping.

- Add broader glyph coverage, fallback fonts, IME composition, and grapheme-aware
  editing.
- Evaluate shaping and kerning support for international chat and player names.
- Replace request-side font oversizing with a consistent sizing contract.
- Evaluate shared metrics for headless layout and windowed drawing.
- Evaluate Go font parsing/rasterization, shaping libraries, and GPU atlas management
  before choosing an in-house renderer. Why? Explained in "Font Renderer" (below)
- Include atlas packing, upload budgets, cache invalidation, and graphics-resource
  lifetime in that evaluation.

A new text renderer is not a prerequisite for every widget above. Language requirements
determine its priority.

### Performance and Visual Validation

Representative warmed headless paths are allocation-tested. Broader profiling remains
planned:

- Profile complete game-sized interfaces, native raylib calls, font uploads, and GPU
  costs.
- Measure long chat histories, large inventories, and frequent market updates.
- Consider ring-buffer graph history if bounded append costs become significant. Current
  bounded appends shift retained points.
- Perform framebuffer review of the new graph, rich tooltip, and nested clipping paths.
  Their initial validation was headless because a display/Xvfb was unavailable.
- Keep application objects long-lived where practical. Do not convert widgets to ECS
  storage without evidence.

### Font Renderer

Background: raylib interprets TTF sizes as pixel height\
(`ascent + descent`) rather than EM units, so faces render much smaller\
than requested — measured per 56px EM: Grenze 0.67x, Open Sans 0.73x,\
Valley Sans 0.82x. At small UI sizes this pushes advances into heavy\
quantization, which reads as uneven spacing, and it punishes light\
weights. The current mitigations are request-side oversizing (see\
`drawTextInContent`) plus font-measured rich-text layout with an\
in-string space derivation (`richSpaceAdvance`) and estimation headless.\
Raylib also ignores GPOS kerning entirely, and headless versus windowed\
metrics remain two paths that merely agree.

Goal: parse, shape, and rasterize glyphs in pure Go, own the texture\
atlas in `render`, and paint textured quads through raylib. That buys\
true EM sizing with no per-font fudge factors, real kerning and shaping,\
one metric path headless and windowed (exact hit-testing and layout in\
tests), and real headless screenshots with text instead of placeholder\
PNGs.

Evaluation notes: `github.com/BaseMax/go-text-render` (MIT) is a\
character-cell layout engine for ASCII, SVG, and PNG output — it does no\
glyph rasterization, textureatlas ownership, or shaping, so on current\
evidence it does not fit this goal; verify before adopting. Better-fitting starting
points to evaluate are `golang.org/x/image/font` with a\
TrueType rasterizer, plus established GPU glyph-cache patterns for atlas\
management.

Scope warnings: atlas packing and texture uploads, cache invalidation,\
CJK coverage, and complex-script shaping if ever needed. This is a\
project, not a patch — sequence it after the widget loops above. Until\
then, keep the request-side oversizing and document per-face effective\
sizes when adding fonts.
