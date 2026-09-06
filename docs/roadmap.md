# rtgui Roadmap — MMORPG Widgets

This roadmap lists widgets `rtgui` should add to support an MMORPG client.
It assumes the current set: `Button`, `Label`, `Checkbox`, `Textbox`,
`ScrollPanel`, `Dropdown`, `Slider`, `ProgressBar`, `Frame`, `TabBar`, plus
the ephemeral UI-owned `Menu` and plain-text `Tooltip`, and instance-owned
`dragdrop.Controller` targets and payloads.

Shipped v1 foundations: `TabBar` ships fixed equal-width tabs without
overflow scroll, closable tabs, or badges. `Menu` ships a flat list with
separators and disabled rows without nested submenus, check items, or
keyboard navigation. `Tooltip` ships single-style text without rarity
colors, stat lines, icons, or item comparison. The P0 entries below keep
the remaining advanced requirements.

## Design Principles for New Widgets

- Compose from existing primitives where possible. Most MMO windows are a
  `Window` containing a `Tabs` control, an `ItemGrid` or `ListView`, and a
  `Tooltip` layer.
- Follow the current ownership model. `widgets` keeps domain data and enabled
  state. `ui.UI` owns hover, press, focus, and popup state. Rendering stays in
  `render.Theme` behind CSS skins.
- Reuse `layout.Node` for bounds and `dragdrop` for inventory and action-bar
  moves. Do not add package-global drag or tooltip state.
- Virtualize long lists. Chat, combat log, guild roster, auction, LFG, and
  damage meters must render only visible rows to keep steady-state simulation
  and packet processing allocation-free.
- Keep each widget skinnable through the existing CSS-to-texture path.
  Unauthored keys stay invisible and unauthored states inherit their base rule.

## P0 — Required to Ship an MMO

- **Window / Dialog**
  - Draggable title bar, close and pin buttons, resizable edges, modal
    blocking, bring-to-front, saved position.
  - Basis for bags, character sheet, map, vendor, settings, mail, auction.
  - Builds on `Frame` plus `ui.UI` focus and modal ownership.

- **Tabs / TabControl** (v1 shipped as `TabBar`; remaining work below)
  - Overflow scrolling, closable tabs, badge counts for unread items.
  - Needed for spellbook, settings, social, auction, bags.

- **ItemSlot and ItemGrid**
  - Slot shows icon, rarity border, stack count, durability pips, bound and
    locked overlays, cooldown dim, and selection highlight.
  - Grid adds grid layout, multi-select, right-click use, and `dragdrop`
    source and target integration with ghost preview.
  - Basis for inventory, bank, loot, trade, vendor, and equipment paperdoll.

- **ActionBar Slot**
  - Special button with icon, keybind label, count, range and mana dim,
    cooldown radial sweep, global-cooldown edge, and interrupt flash.
  - Needs a cooldown manager separate from the current `value` field.

- **ResourceBar**
  - Extends `ProgressBar` with segmented ticks, delayed-damage ghost,
    text overlay, color ramp by percent, rested bonus zone, and castbar mode
    with interruptible flag and latency ticks.
  - Covers health, mana, stamina, experience, reputation, and cast bars.

- **Buff and Debuff Strip**
  - Wrapping flow of icons with time sweep, stack count, border color by
    dispel type, hover tooltip, right-click cancel, and sort by time or
    priority.
  - Used for player auras, target auras, and party raid buffs.

- **ChatBox: Log, Tabs, and Input**
  - Virtualized rich-text log with channels, filters, history limit, and
    clickable item links and player names.
  - Unread flashing tabs plus multi-line input with history, whisper
    autocomplete, and slash-command prefix.
  - Current `Textbox` is single-line bounded storage and is not enough here.

- **Rich Tooltip Manager** (plain-text v1 shipped as `Tooltip`; remaining work below)
  - Hover delay, follow-mouse with screen clamping, rarity title, stat lines,
    `Shift` to compare against equipped item, embedded icons, and talent
    and quest variants.
  - Model as hover-owned popup state in `ui.UI`, similar to the dropdown
    popup path.

- **ContextMenu / PopupMenu** (flat v1 shipped as UI-owned `Menu`; remaining work below)
  - Nested submenus, check items, separators, disabled items, keyboard
    navigation.
  - Needed for right-click on players, slots, chat lines, and unit frames.

- **ListView / TableView**
  - Virtualized rows, sortable columns, single and multi-select, alternating
    rows, icons, and right-aligned numbers.
  - Basis for quest log, guild roster, friends, LFG, mail, auction, damage
    meter, and combat log.

- **Toast / Notification Feed**
  - Stacked queue with timer bar, click-through action, and pooled allocation.
  - Used for loot, quest accept and complete, achievement, level-up, and
    group invite.

## P1 — Expected MMO Chrome

- **Unit Frames: Player, Target, Party, Raid, Boss**
  - Portrait, health and mana bars, castbar, buff strip, leader and loot
    markers, role and PvP flags, aggro highlight, ready-check overlay.
  - Compose from `ResourceBar` plus portrait frame.

- **Minimap, World Map, and Compass**
  - Circular mask, zoom and rotate, POI and marker and ping layer,
    fog-of-war, player position and facing cone, zone text.
  - Full map reuses the same marker layer with pan and zoom.

- **Quest Tracker / Objective List**
  - Collapsible header, distance-sorted objectives, click-to-ping, progress
    shimmer on update.

- **Settings Controls: ToggleSwitch, RadioGroup, SpinBox, Scrollbar**
  - Settings and keybind UI cannot be built from `Checkbox`, `Slider`, and
    `Dropdown` alone.
  - Add an explicit keybind-capture button with a `Press a key` state.

- **TreeView / Skill Tree**
  - Talent nodes with connectors and available, invested, and locked states,
    plus hover preview.
  - Also covers quest categories and crafting recipe trees.

## P2 — Nice to Have

- Radial menu for gamepad-friendly selection.
- Damage meter bars with per-row sparkline.
- Mailbox, two-sided trade window with lock and accept flow, loot-roll frame.
- Guild bank with permission states.
- Calendar and event sign-up list.
- Color picker for tabard and guild emblem editing.
- Nameplate pool for world-space overhead health bars.
- Scrolling combat text layer.

## Suggested Build Order

1. `Window` — unblocks every other window (`TabBar`, `Menu`, and plain
   `Tooltip` v1 foundations already shipped).
2. `ItemSlot` and `ItemGrid`, `ActionSlot`, `ResourceBar`, `BuffStrip` —
   playable combat and inventory loop.
3. Virtualized `ListView`, `ChatBox`, `Toasts` — social and info loop.
4. Minimap markers, `UnitFrames`, quest tracker — full MMO feel.
