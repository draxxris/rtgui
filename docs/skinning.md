# Skinning

CSS files select widget looks. Every pixel requires a CSS-to-texture pathway:
unauthored keys stay invisible (text still draws) and unauthored states inherit
their base rule. Parse and mapping stay headless in `skin`; texture upload and
registry writes live in `render`.

## Selectors

Selectors use `Kind[.class][::part][:pseudo]` plus `*`. Exact case. Examples:
`Button`, `Button.accent`, `Frame.titled-titlebar:focus`, `Slider::thumb`,
`Tooltip.item`, `ProgressBar::fill`.

Supported properties: `border-image-source`, `border-image-slice`,
`background-image`, `background-image-tint`, `background-color`,
`border-image-source-tint`, `border-radius`, `padding`, `color`, `font-size`,
`font-family`, `font-italic-family`. Anything else is a hard error naming the
selector and declaration.

## Background layers

`background-image` accepts `url(...)`, `none`, or one to four comma-separated
gradient layers with the first layer on top. It also accepts one leading
`url(...)` texture followed by up to four gradient layers, for example:

```css
Button.quest-action {
    background-image: url("quest/button_surface_v2.png"),
        radial-gradient(circle at 50% 12%, #ffffff44, #ffffff00 62%),
        linear-gradient(to bottom, #3b5373dd, #141c28ee);
}
```

The mixed form is intentionally constrained: the URL must be first, there may
be only one URL, and `none` cannot be mixed with other layers. The first item
is the topmost layer; rendering composites the gradients back-to-front and
then paints the leading texture, preserving CSS order. `none` drops inherited
textures and gradients.

Each layer is either linear or radial with 2–4 color stops:

- `linear-gradient([to <direction> | <angle>deg,] <color> [<pos>%], ...)`
  Directions: `to bottom` (default), `to top`, `to right`, `to left`, and the
  four `to <vertical> <horizontal>` diagonals. Angles follow CSS: `0deg`
  points up and values grow clockwise.
- `radial-gradient([circle [at <x>% <y>%],] <color> [<pos>%], ...)`
  The center defaults to `50% 50%`; distance 1 is the farthest corner.

Missing stop positions follow CSS Images 3: a missing first stop anchors at
`0%`, a missing last at `100%`, and interior runs spread evenly between
explicit neighbors. Out-of-order positions clamp forward into hard stops.

A four-layer shell with a highlight, an angled mid fade, a base, and a bottom
vignette:

```css
Frame.card {
    background-color: #0b1524;
    background-image: radial-gradient(circle at 30% 20%, #3a5a7a80, #00000000 60%),
        linear-gradient(135deg, #17b978 0%, #0ea071 50%, #086972 100%),
        linear-gradient(to bottom, #2b3d54 0%, #16263c 45%, #0b1524 100%),
        linear-gradient(to top, #00000066, #00000000 30%);
    border-radius: 6;
    padding: 8;
}
```

## Tooltip classes

`Tooltip` is the base shell for every tooltip. `Tooltip.<class>` variants
scope special presentations: a rich tooltip carrying `Class: "item"` draws the
`Tooltip.item` shell. The gallery item tooltip uses a radial highlight over a
three-stop vertical base:

```css
Tooltip.item {
    background-color: #0b1524;
    background-image: radial-gradient(circle at 30% 20%, #3a5a7a80, #00000000 60%),
        linear-gradient(to bottom, #2b3d54 0%, #16263c 45%, #0b1524 100%);
    border-image-source: url("kenney/panel_border_grey.png");
    border-image-slice: 6;
    border-radius: 6;
    padding: 8;
}
```

Set the class on the payload: `core.RichTooltip{Title: "Thunderfury",
Class: "item", ...}`. Class changes bust the tooltip layout cache; steady
hover and draws allocate nothing after warmup.

## Inheritance

Later rules for the same key win per field. States inherit omitted fields
from their same-kind, same-part normal rule — including the whole gradient
stack. An explicit `background-image` replaces every inherited layer, so a
state that sets one gradient never keeps the normal rule's other layers. A
mixed leading-URL declaration replaces the inherited stack with its texture
and gradients as one unit. `background-image` also travels as a unit across
specificity: a class that declares gradients drops the base texture, and one
that declares a texture drops base gradients — a texture never silently buries
class gradients.
`border-radius` clips background layers to a rounded rectangle and never
reshapes the border texture itself.

## Performance notes

Two-stop edge-to-edge fills draw as one quad. Cardinal multi-stop fills slice
into exact strips. Angled multi-stop fills cap at 128 tessellation rows and
radial fills at a 12x12 grid per frame; keep radial and angled multi-stop
fills on small widgets such as buttons and tooltips rather than fullscreen
panels. Headless tests do not establish GPU performance.
