# List Contract

Collapsible category list matching the auction-house reference: icon and label
category rows with a right-edge chevron, one expanded category with indented
leaf rows, single gold-highlighted leaf selection, dark pane. Category rows
are toggle-only and never selectable. The `Sell Your Item` button in the
reference is a separate `Button`, not part of the list.

## Decision: new WidgetKind

`List` is a new `core.WidgetList` kind (like `TabBar`/`Dropdown`), not a
builder of buttons. Per-row hit testing, internal scrolling, and stable-ID
selection need one widget with cached visible rows. A button per row would
explode the registry and defeat windowed drawing.

## Structure

New files: `widgets/list.go`, `render/list.go`, `ui/list.go`.

```go
type ListItem struct {
    ID       string
    Label    string
    Icon     string // inline-icon whitelist name, "" = none
    Expanded bool
    Children []ListItem // empty = selectable leaf
}

type ListRow struct {
    ID, Label, Icon string
    Depth           int
    HasChildren     bool
    Expanded        bool
}

func NewList(name string, bounds core.Rect) *List
```

`SetItems` deep-copies; `Items` and `VisibleRows` return defensive copies;
`AppendVisibleRows` reuses caller storage. Selection is a leaf ID (`""` when
unset) and survives `SetItems` only when the ID remains a leaf. Expanded
flags come from the incoming tree. All `Set*` domain mutations return `bool`
change reports; geometry and styling setters return `*List` for chaining.

## Layout

Leaf widget with fixed row height (default 28) and per-depth indent
(default 16). `Theme.ListContent` is the single skin-aware viewport for
drawing, hit testing, and scroll bounds; the widget reconciles its derived
`MaxScroll` against that height via `EnsureScrollBounds`. Rows lay out in
`render.ListRowsContent`, the viewport minus the scrollbar track when
overflowing, so draw and hit-test geometry agree and chevrons never slide
under the track. Only visible rows draw or hit, so long lists cost one
window, not one widget per row. Chevron zone is the right 28px of category
rows; icon slot is 20px when `Icon` is set. `Text` resolves the selected
label from the item tree, so it survives collapsing the parent category.

## Behavior

Pointer press arms the list; release on a category row toggles it (fires
`OnListToggle` on real change, then `OnClick`), release on a leaf selects it
(fires `OnListSelect` on real ID change, then `OnClick`). Same-row re-clicks
fire `OnClick` only. Releases on the scrollbar or outside rows consume
without mutation. Wheel, thumb drag, track jump, and hover share one
scrollbar gesture path with scroll panels (`scrollState` helpers, one UI
owner pair); the innermost overflowing container owns each gesture.
`SelectListItem` and `SetListExpanded` run the same commit path headless
(also exposed through `sim.Stage`); `Activate` re-fires `OnClick` for the
current selection and rejects an empty one. Disabled lists block pointer and
semantic paths.

## Callbacks

`widgets.Callbacks` carries `ListSelect func(string)` and
`ListToggle func(string, bool)` under the single-registry replace rule.
Named setters are `u.OnListSelect` and `u.OnListToggle`; fluent setters are
`list.OnSelect` and `list.OnToggle`.

## Skinning

No new `SkinPart`. Whole-widget `Background`/`Border`/padding plus label
`color`/`font-size` come from `List` rules; selected and hovered fills come
from `List::highlight` (`:selected` for selection); the scrollbar reuses
`List::track` and `List::thumb`. Chevrons are geometry-only triangles (right
means collapsed, down means expanded), like textbox carets, so they never
need a texture; `List::arrow` is rejected loudly. Icons resolve at draw time
from the inline-icon whitelist; unknown names log a placeholder without
pixels. Selected rows add a fixed gold accent tick plus a subtle divider.

```css
List { background-image: url("kenney/background.png"); background-image-tint: #1a1d24;
  border-image-source: url("kenney/panel_border_grey.png"); border-image-slice: 8;
  padding: 4; color: #c9b896; font-size: 16; }
List::highlight { background-image: url("kenney/background.png"); background-image-tint: #3a2f1d; }
List::highlight:selected { background-image: url("kenney/background.png"); background-image-tint: #4a3a20; }
```

## Gallery and tests

`gallery.css` carries the rules above; the headless smoke exercises toggle,
select, and draw through the facade. The windowed gallery shows a floating
Categories window (slot-table placed, layout-test locked) with the
reference fixture — Materials expanded, Essences preselected — plus a Sell
button reporting into the status line. Widget, render geometry and draw,
skin vocabulary, UI pointer and semantic, gallery layout, window demo, CSS,
and smoke tests cover the widget; `mise run complexity` stays under 15.

## Non-goals

Multi-select, check rows, sortable or filterable columns, drag-reorder,
variable row heights, per-row rich runs, horizontal scrolling, keyboard
traversal, and persistence. The virtualized `TableView` stays on the roadmap.
