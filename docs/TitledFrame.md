# TitledFrame Contract Proposal

## Goal

Reusable quest-log / spellbook-style window: a titlebar with a title on the
left and an `X`/close button on the right, plus an open content area below
for caller-owned widgets.

## Decision: composition, not a new WidgetKind

No new `core.WidgetKind`, no new `render.Theme` draw path. `TitledFrame` is a
`widgets`-package builder owning five existing widgets, wired through the
existing ownership tree (`layout.Node.AddChild` + `SetPoint`, WoW FrameXML
style as in `examples/widget_gallery/layout_spec.go`). The roadmap already
says to compose game panels from reusable widgets rather than adding a type
per screen; ownership hands us clipping, inherited availability, subtree
`BringToFront`, and hit-testing for free.

The reason for a builder (rather than a `Widget`) is structural, not the
import graph: one `Widget` is one name plus one `layout.Node`, and this
control needs five of each. This is the first exported non-`Widget` type in
`widgets/`. Builders stay `ui`-free and hand back `[]Widget`; the caller
registers them. The tree is built with `AddChild` on the layout nodes after
`Add` (both orderings are blessed by `docs/interaction.md`; this contract
uses post-`Add` `AddChild` inside `Layout`, matching the gallery's
`SetParent`-after-`Add` outcome).

## Structure

New file: `widgets/titled_frame.go`.

```go
type TitledFrame struct {
    root     *Frame   // name as given, e.g. "questLog"
    titleBar *Frame   // name + "/titlebar"
    title    *Label   // name + "/title"
    close    *Button  // name + "/close", text "X"
    content  *Frame   // name + "/content", caller parents into this
    onClose  func()
}

func NewTitledFrame(name string, bounds core.Rect, title string) *TitledFrame
```

Child names use the `name + "/..."` convention so registry identities are
predictable (`facade.Lookup("questLog/content")`) and unlikely to collide
with flat names (`primaryButton`, `demoFrame`). This is a convention, not a
namespace: `u.widgets` is flat, so a literal `questLog/content` widget could
still collide. Hierarchical keys already exist (`item/iron-plate`); nothing
in `ui/` splits names on `/`.

`TitledFrame` is not a `widgets.Widget`. Callers register the owned widgets,
then call one layout entry point:

```go
tf := widgets.NewTitledFrame("questLog", core.Rect{}, "Quest Log")
facade.Add(tf.Widgets()...)
tf.Layout() // AddChild + points + layout.Arrange(root.Frame(), core.Rect{}) in one call
facade.Add(questText, acceptBtn, declineBtn...)
facade.SetParent("questText", tf.ContentName())
tf.Arrange() // re-arrange after content children join, mirroring applyFrameMove
```

`ui.Remove("questLog")` disposes all five widgets via `removeTree` and zeroes
their callbacks, leaving the builder holding dangling pointers. Removal
invalidates the builder; do not reuse it after remove.

## API

```go
func (t *TitledFrame) Widgets() []Widget  // [root, titleBar, title, close, content] in z-order
func (t *TitledFrame) Layout() error      // AddChild + points + Arrange in one call
func (t *TitledFrame) Arrange() error     // layout.Arrange(root.Frame(), core.Rect{})

func (t *TitledFrame) ContentName() string // for facade.SetParent(child, ...)
func (t *TitledFrame) Content() *Frame
func (t *TitledFrame) CloseButton() *Button

func (t *TitledFrame) Title() string
func (t *TitledFrame) SetTitle(s string) bool
func (t *TitledFrame) SetTitleBarHeight(h float32) *TitledFrame // configure before Layout
func (t *TitledFrame) SetBorderInset(inset float32) *TitledFrame // configure before Layout
func (t *TitledFrame) SetVariant(style string) *TitledFrame
func (t *TitledFrame) OnClose(fn func()) *TitledFrame

func (t *TitledFrame) Show()
func (t *TitledFrame) Hide()   // hides, no callback
func (t *TitledFrame) Toggle() // IsOpen() ? Close() : Show()
func (t *TitledFrame) IsOpen() bool
func (t *TitledFrame) Close()  // Hide() + onClose callback
```

Dropped from the earlier draft: `Name()` and `Root()` (the caller already
holds the name; `Arrange()` covers the root) and the five `Set*Class`
methods, collapsed into `SetVariant`. `SetVariant("guild")` derives
`guild-frame` / `guild-titlebar` / `guild-title` / `guild-close` /
`guild-content`; the default variant is `"titled"`, byte-identical to the
table below. `Content()` stays for tests and direct layout access;
`ContentName()` is the `SetParent` handle; `CloseButton()` is the styling
handle.

Setter conventions: `SetTitle` returns `bool` (matches `base.SetText`);
geometry and variant setters return `*TitledFrame` for chaining.
`SetTitleBarHeight` / `SetBorderInset` / `SetVariant` only store values;
after `Layout()`, call `Layout()` again to apply them. Configure them
before the first `Layout()`.

## Layout

- `const DefaultTitleBarHeight = 32`, overridable via `SetTitleBarHeight`.
- `SetBorderInset(inset)` offsets the titlebar `TopLeft` / `TopRight` and the
  content `BottomRight` inward so children clear the root's 9-patch ring.
  CSS `padding` on a `Frame` only moves its own text via `controlContentRect`,
  never children, so this cannot come from the theme (`widgets` cannot see
  it). Document the value alongside the CSS and keep it equal to the root's
  `border-image-slice`.
- `titleBar`: `TopLeft -> root.TopLeft` (+ inset), `TopRight ->
  root.TopRight` (- inset), fixed height.
- `close`: 24x24, `TopRight -> titleBar.TopRight` with 4px inset. Text `X`,
  tooltip `Close`. The only opaque surface in the chrome.
- `title`: fills `titleBar` minus the close reservation (left inset 10px,
  right inset 32px), vertically centered.
- `content`: `TopLeft -> titleBar.BottomLeft`,
  `BottomRight -> root.BottomRight` (- inset). Empty `Frame`; caller owns its
  children.
- Hit transparency: the constructor sets `SetInputTransparent(true)` on
  `titleBar`, `title`, and `content`. `hitNode` descends into children before
  testing a parent surface, so the close button and content children still
  hit, while empty chrome falls through to the root. The root is therefore
  the `activeFrame` for clicks anywhere on empty window area, which makes
  `Frame.titled-frame:focus` reachable for a whole-window glow and keeps the
  door open for a later press-drag move. (Without this, `titleBar` and
  `content` tile the root and the root `:focus` rule would be dead, since
  only the innermost `Frame` ancestor becomes `activeFrame`.)
- Clipping is free: `drawNode` intersects `childClip` and `hitNode` will not
  descend outside it, so ownership-parented content children clip correctly
  with no extra work.
- Moving the root with `layout.MoveFrame` moves children (same pattern as the
  gallery `demoFrame` + `frameChildButton`).

## Behavior

- `Close()` always hides first, then fires `onClose` when set. Letting a
  callback opt out of hiding would break `IsOpen()` and the reopen path.
- `Hide()` hides with no callback. `Toggle()` is `IsOpen() ? Close() :
  Show()`. Programmatic close and the `X` button share the `Close()` path.
- Register close interest only via `tf.OnClose(...)`. Never use
  `facade.OnClick("questLog/close", ...)` for this: `u.OnClick` replaces the
  callback slot and silently detaches `Close()`.
- `Enabled`/`Visible` delegate to root; `ui.available()` already checks
  ancestors, so disabling the root disables the subtree.
- Stacking is app-side: open calls `facade.BringToFront("questLog")`, same as
  any root. No built-in modal blocking.
- Title text, align, and font setters delegate to the title `Label`, matching
  the fluent setters on `Button`/`Label`.

## CSS Skinning

Skinning is class variants on the composed widgets. Theme lookup order is
`*` -> `Kind` -> `.class` -> `Kind.class`, with
`:hover` / `:active` / `:focus` / `:disabled` inheriting the base rule.
An unauthored class falls back to the base `Frame` / `Button` / `Label` rule;
an unauthored kind stays invisible with text still drawn.

Default variant `"titled"` (set by the constructor, changed via
`SetVariant`):

| Sub-widget | Kind | Class | Selectors |
|---|---|---|---|
| root | Frame | `titled-frame` | `Frame.titled-frame`, `Frame.titled-frame:focus`, `Frame.titled-frame:disabled` |
| titlebar | Frame | `titled-titlebar` | `Frame.titled-titlebar` |
| title | Label | `titled-title` | `Label.titled-title` |
| close | Button | `titled-close` | `Button.titled-close`, `Button.titled-close:hover`, `:active`, `:focus`, `:disabled` |
| content | Frame | `titled-content` | `Frame.titled-content` |

The root `:focus` rule is the whole-window glow; it is reachable because the
chrome surfaces are input-transparent (see Layout). Universal
`.titled-close:hover` works; `Button.titled-close:hover` wins by
specificity. All five use whole-widget rules; no new `::part` is needed
(`Frame` / `Button` / `Label` expose no allowed parts and that allowlist
stays untouched). The close glyph is button text `X` skinned with
`color` / `font-size`, not an icon part.

Supported properties are the existing CSS subset:

- `Frame.titled-frame` / `-titlebar` / `-content`: `background-color`,
  `background-image` + `background-image-tint`, `linear-gradient(...)`,
  `border-image-source` + `border-image-slice` +
  `border-image-source-tint`, `border-radius`, `padding`.
- `Label.titled-title`: `color`, `font-size`, `font-family`,
  `font-italic-family` (plus background/border/padding for a title pill).
- `Button.titled-close`: full `Button` set (`background-color`,
  `background-image`, `border-image-*`, `border-radius`, `padding`, `color`,
  `font-size`).

Two overlay caveats (`SkinDescriptor.Overlay` is additive per field):

- A `background-color` on titlebar/content is painted under the inherited
  base `Frame` texture. Titlebar and content rules must start with
  `background-image: none;` for their fill to show.
- Without an opt-out, titlebar and content inherit the base `Frame` 9-patch
  ring. This change adds `border-image-source: none` to the LOOK subset
  (parsed as `NoTexture`, which drops the inherited ring through `Overlay`
  while keeping slice insets); the titlebar/content rules below depend on
  it. "Zero engine change" does not hold here.

Gallery additions to `testdata/skins/gallery.css`:

```css
Frame.titled-frame { border-image-source: url("kenney/panel_border_grey.png"); border-image-slice: 8; padding: 8; }
Frame.titled-frame:focus { border-image-source: url("kenney/blue/button_rectangle_border.png"); border-image-source-tint: #e1f2ff; }
Frame.titled-titlebar { background-image: none; background-color: #24354c; border-image-source: none; padding: 4; }
Label.titled-title { color: #e6f0ff; font-size: 20; }
Frame.titled-content { background-image: url("kenney/background.png"); background-image-tint: #1c2740; border-image-source: none; padding: 8; }
Button.titled-close { background-color: #5c1d24; border-image-source: url("kenney/red/button_rectangle_border.png"); border-image-slice: 8; }
Button.titled-close:hover { background-color: #8b2635; }
Button.titled-close:active { background-color: #3b1015; }
```

A dedicated `TitledFrame { ... }` kind selector is a non-goal for v1: it
needs a new `core.WidgetKind`, a `kindSelectors` entry, and a `render` draw
path. Variant classes reskin quest-log / spellbook windows without one.

## Gallery Demo

- New `questButton` (`Open Quest Log`) in the left panel slot table.
- New `questLog` `TitledFrame` (~420x320, screen root, centered), initially
  hidden via `Hide()`.
- Content children parented to `questLog/content`: quest body `Label` /
  `RichText` plus `Accept` / `Decline` buttons side-by-side at the content
  bottom. Both are non-functional by design: `OnClick` only calls
  `setStatus("Quest accepted (demo)")` / `("Quest declined (demo)")` and the
  window stays open. Callers must keep children inside content bounds; an
  unbounded quest body overflows visibly, so long text should use a
  `ScrollPanel` content host (drawer-based, not child widgets) or accept
  truncation. The demo uses short fixed text.
- `X` closes (`Close()`); `questButton` re-shows plus `BringToFront`.

## Tests

- `widgets/titled_frame_test.go`: unique `/`-suffixed names, title init,
  close text `X`; `Layout` geometry (titlebar on top with border inset, close
  inside top-right, content fills remainder); chrome input-transparency;
  `SetTitle`; `Show` / `Hide` / `Toggle` / `Close` (callback fires after
  hide); `SetVariant` class derivation; `SetBorderInset` offsets;
  content child follows `MoveFrame`.
- `skin` parse test: the five selectors plus pseudos parse, including
  `border-image-source: none`.
- `widgets` class test: default variant sets classes, `SetVariant`
  propagates to `Snapshot().Class` so `Theme.Lookup` resolves the descriptor.
- Gallery headless smoke (`-frames 3 -screenshot`): open, Accept/Decline
  update status, `X` hides, reopen works. `mise run complexity` stays under 15.

## Non-Goals

No modal input blocking, no resize handle, no persistence, no new skin
texture pipeline. Titlebar drag-move stays a non-goal: `Frame` / `Label`
expose no pressable surface today (the title `Label` covers most of the
titlebar), so a future drag needs engine-side gesture work, not just layout.
The input-transparent title is placed now so that work has one fewer blocker.
