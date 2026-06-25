# Charm v2 (charm.land/*) API Reference — Nord Theming, Mouse, Responsive Layout

> Source-verified against the pinned module cache: `bubbletea/v2@v2.0.1`, `bubbles/v2@v2.0.0`,
> `lipgloss/v2@v2.0.0`, `huh/v2@v2.0.0-20260226141913-a8934362ea3b`. These mirror
> `github.com/charmbracelet/*` v2. Kept as the authoritative reference for the TUI rewrite.

**Biggest v2 change:** `WithAltScreen`, `WithMouseCellMotion`, `WithMouseAllMotion` no longer exist
as `tea.NewProgram` options. AltScreen, mouse mode, background color, window title, and cursor are
now **fields on the `tea.View` value returned by `View()`**. `Model.View()` returns `tea.View`.

## Nord palette
```go
package theme
import "charm.land/lipgloss/v2"
var (
    Nord0 = lipgloss.Color("#2E3440") // Polar Night (bg)
    Nord1 = lipgloss.Color("#3B4252")
    Nord2 = lipgloss.Color("#434C5E")
    Nord3 = lipgloss.Color("#4C566A")
    Nord4 = lipgloss.Color("#D8DEE9") // Snow Storm (fg)
    Nord5 = lipgloss.Color("#E5E9F0")
    Nord6 = lipgloss.Color("#ECEFF4")
    Nord7  = lipgloss.Color("#8FBCBB") // Frost
    Nord8  = lipgloss.Color("#88C0D0")
    Nord9  = lipgloss.Color("#81A1C1")
    Nord10 = lipgloss.Color("#5E81AC")
    Nord11 = lipgloss.Color("#BF616A") // Aurora red
    Nord12 = lipgloss.Color("#D08770") // orange
    Nord13 = lipgloss.Color("#EBCB8B") // yellow
    Nord14 = lipgloss.Color("#A3BE8C") // green
    Nord15 = lipgloss.Color("#B48EAD") // purple
)
```

## lipgloss/v2 colors & layout
- `lipgloss.Color("#88C0D0")` **still valid**; returns stdlib `image/color.Color`. `Foreground`/
  `Background` take `color.Color` (so `color.RGBA{}` also works). Bad parse → silent `NoColor{}`.
- No `AdaptiveColor`/`CompleteColor` structs. Now functions:
  ```go
  ld := lipgloss.LightDark(isDark)     // isDark from tea.BackgroundColorMsg.IsDark()
  fg := ld(theme.Nord0, theme.Nord6)   // light→Nord0, dark→Nord6
  cf := lipgloss.Complete(profile)     // cf(ansi, ansi256, truecolor)
  ```
  Get `isDark` via `tea.RequestBackgroundColor()` → handle `tea.BackgroundColorMsg` (`.IsDark()`).
- Layout: `JoinHorizontal(pos, ...)`, `JoinVertical(pos, ...)`, `Place(w,h,hPos,vPos,str,opts...)`,
  `PlaceHorizontal/PlaceVertical`. Positions: `Top=0 Bottom=1 Center=.5 Left=0 Right=1`.
  Style sizing: `Width/Height/MaxWidth/MaxHeight/Align/AlignHorizontal/AlignVertical`.
  `Place` whitespace opts: `WithWhitespaceStyle(s)`, `WithWhitespaceChars(s)` (color the padding).

### Whole-screen background — two ways (prefer #1)
```go
// 1. terminal-level, fills EVERY cell (OSC):
v := tea.NewView(m.layout()); v.AltScreen = true; v.BackgroundColor = theme.Nord0
// 2. content-level: top style Background+Width(m.width).Height(m.height), or
//    lipgloss.Place(w,h,...,WithWhitespaceStyle(NewStyle().Background(Nord0)))
```
`lipgloss.Canvas`/`Layer` compositor exists for z-ordered overlays (`NewCanvas(w,h)`,
`NewLayer(content).X().Y().Z()`, `canvas.Render()`) — but bubblezone may not cooperate with it.

## bubbles/v2 list — theme BOTH chrome (`Styles`) and rows (delegate)
```go
type Styles struct {
    TitleBar, Title, Spinner lipgloss.Style
    Filter textinput.Styles                 // nested!
    DefaultFilterCharacterMatch lipgloss.Style
    StatusBar, StatusEmpty, StatusBarActiveFilter, StatusBarFilterCount lipgloss.Style
    NoItems, PaginationStyle, HelpStyle lipgloss.Style
    ActivePaginationDot, InactivePaginationDot, ArabicPagination, DividerDot lipgloss.Style
}
type DefaultItemStyles struct {
    NormalTitle, NormalDesc, SelectedTitle, SelectedDesc,
    DimmedTitle, DimmedDesc, FilterMatch lipgloss.Style
}
```
- Build chrome from `list.DefaultStyles(true)`; rows from `list.NewDefaultItemStyles(true)`.
- `DefaultDelegate{ShowDescription, Styles, UpdateFunc, ShortHelpFunc, FullHelpFunc}`;
  `(*DefaultDelegate).SetHeight(i)` / `.SetSpacing(i)` (height ignored when `ShowDescription==false`,
  then height is always 1). Apply with `l.SetDelegate(d)`; chrome via `l.Styles = s`.
- **Mouse row math:** stride = `delegate.Height() + delegate.Spacing()`;
  `index = Paginator.Page*Paginator.PerPage + relativeY/stride` (after subtracting top chrome).
  Prefer bubblezone-per-item to avoid this.

## bubbles/v2 textinput & textarea — use `Styles()` / `SetStyles()`
```go
// textinput.Styles{ Focused, Blurred StyleState; Cursor CursorStyle }
// StyleState{ Text, Placeholder, Suggestion, Prompt lipgloss.Style }
// CursorStyle{ Color color.Color; Shape tea.CursorShape; Blink bool; BlinkSpeed time.Duration }
s := ti.Styles(); s.Focused.Text = s.Focused.Text.Foreground(Nord6); s.Cursor.Color = Nord8
s.Cursor.Shape = tea.CursorBar; ti.SetStyles(s)

// textarea.Styles{ Focused, Blurred StyleState; Cursor CursorStyle }
// StyleState{ Base, Text, LineNumber, CursorLineNumber, CursorLine, EndOfBuffer, Placeholder, Prompt }
t := ta.Styles(); t.Focused.Base = t.Focused.Base.Background(Nord0)
t.Focused.CursorLine = t.Focused.CursorLine.Background(Nord1); ta.SetStyles(t)
```
Cursor shapes: `tea.CursorBlock | tea.CursorUnderline | tea.CursorBar`. No "selected text" style field.

## spinner / progress / viewport / help / key
```go
sp := spinner.New(); sp.Spinner = spinner.Dot; sp.Style = lipgloss.NewStyle().Foreground(Nord8)

// progress: public fields FullColor, EmptyColor, ShowPercentage, PercentageStyle; options:
//   WithColors(...color.Color) (gradient if >1), WithColorFunc, WithFillCharacters, WithWidth, WithoutPercentage
p := progress.New(progress.WithColors(Nord10, Nord8), progress.WithWidth(40))
p.FullColor = Nord8; p.EmptyColor = Nord2; p.PercentageStyle = lipgloss.NewStyle().Foreground(Nord4)

// viewport: Style, HighlightStyle, SelectedHighlightStyle, StyleLineFunc
vp.Style = lipgloss.NewStyle().Background(Nord0).Foreground(Nord6)

// help.Styles{ Ellipsis, ShortKey, ShortDesc, ShortSeparator, FullKey, FullDesc, FullSeparator }
h := help.New(); h.Styles.ShortKey = lipgloss.NewStyle().Foreground(Nord8)

// key: logical only — key.NewBinding(key.WithKeys("k","up"), key.WithHelp("↑/k","up")); colors come from help.Styles
```

## huh/v2 — custom theme
**`huh.Theme` is an INTERFACE** (`Theme(isDark bool) *Styles`); concrete type is **`huh.Styles`**.
Build `*huh.Styles` (start from `huh.ThemeBase(isDark)`), wrap in `huh.ThemeFunc`, pass to
`form.WithTheme(huh.Theme)`.
```go
type Styles struct {
    Form FormStyles; Group GroupStyles; FieldSeparator lipgloss.Style
    Blurred, Focused FieldStyles; Help help.Styles
}
// FieldStyles has: Base, Title, Description, ErrorIndicator, ErrorMessage, SelectSelector, Option,
//   NextIndicator, PrevIndicator, Directory, File, MultiSelectSelector, SelectedOption,
//   SelectedPrefix, UnselectedOption, UnselectedPrefix, TextInput(TextInputStyles),
//   FocusedButton, BlurredButton, Card, NoteTitle, Next
// TextInputStyles{ Cursor, CursorText, Placeholder, Prompt, Text }
func NordHuhTheme() huh.Theme { return huh.ThemeFunc(func(isDark bool) *huh.Styles {
    t := huh.ThemeBase(isDark)
    t.Focused.Title = t.Focused.Title.Foreground(Nord8).Bold(true)
    t.Focused.Base = t.Focused.Base.BorderForeground(Nord8)
    t.Blurred = t.Focused; t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
    return t
})}
```

## Mouse in bubbletea v2
```go
// All implement tea.MouseMsg (interface{ fmt.Stringer; Mouse() Mouse }):
//   MouseClickMsg, MouseReleaseMsg, MouseWheelMsg, MouseMotionMsg (all are `Mouse` underneath)
// Mouse{ X, Y int; Button MouseButton; Mod KeyMod }  — X,Y zero-based, (0,0)=top-left
// Buttons: MouseNone, MouseLeft, MouseMiddle, MouseRight, MouseWheelUp/Down/Left/Right,
//          MouseBackward, MouseForward, MouseButton10/11
// Enable via View, NOT program option:
//   MouseModeNone | MouseModeCellMotion | MouseModeAllMotion
v := tea.NewView(m.render()); v.AltScreen = true; v.MouseMode = tea.MouseModeCellMotion
```
```go
case tea.MouseClickMsg:
    e := msg.Mouse() // or msg.X/msg.Y directly
    if e.Button == tea.MouseLeft { m.handleClick(e.X, e.Y) }
case tea.MouseWheelMsg:
    switch msg.Button { case tea.MouseWheelUp: m.list.CursorUp(); case tea.MouseWheelDown: m.list.CursorDown() }
```
Hit-testing: native `View.OnMouse func(MouseMsg) Cmd` hook, OR **bubblezone v2** (recommended).

### bubblezone IS v2-ready — `github.com/lrstanley/bubblezone/v2 v2.0.0`
```go
zone.NewGlobal(); defer zone.Close()                     // in main, before Run()
// View(): wrap clickable regions, Scan the OUTER frame:
row := zone.Mark("item-"+strconv.Itoa(i), m.renderRow(it))
v := tea.NewView(zone.Scan(body)); v.AltScreen = true; v.MouseMode = tea.MouseModeCellMotion
// Update():
case tea.MouseClickMsg:
    if msg.Button == tea.MouseLeft {
        if zone.Get("item-"+strconv.Itoa(i)).InBounds(msg) { m.selected = i }
    }
// ZoneInfo{ StartX,StartY,EndX,EndY }; .InBounds(tea.MouseMsg), .Pos(msg)(x,y), .IsZero()
```
**Caveat:** bubblezone may misbehave with the lipgloss v2 canvas/compositor; for plain
Join* layouts it works fine.

## Cursor — real terminal cursor via `View.Cursor *tea.Cursor`
```go
// tea.Cursor{ Position{X,Y}; Color color.Color; Shape CursorShape; Blink bool }; tea.NewCursor(x,y)
// textinput/textarea Cursor() *tea.Cursor returns nil unless VirtualCursor()==false AND focused.
m.textarea.SetVirtualCursor(false); m.textarea.Focus()
// in View(), offset by the component's on-screen position:
if c := m.textarea.Cursor(); c != nil { c.Position.X += offX; c.Position.Y += offY; v.Cursor = c }
```
Keep default virtual cursor (drawn into component output, colored by `Styles.Cursor.Color`) → leave
`View.Cursor` unset. NOTE: some bubbles doc comments still say `tea.NewFrame`/`f.Cursor`; in v2.0.1
it is `tea.View`/`tea.NewView`/`View.Cursor`.

## Gotchas summary
1. `View()` returns `tea.View` (`tea.NewView(s)`). 2. AltScreen/mouse/bg/title/cursor are View
fields. 3. `tea.MouseMsg` is an interface. 4. textinput/textarea use `SetStyles()/Styles()`; list
public `Styles`; spinner/viewport public `Style`. 5. `lipgloss.Color("#hex")` still works; adaptive
colors are funcs. 6. huh: `*huh.Styles` + `huh.ThemeFunc`, start from `huh.ThemeBase(isDark)`.
7. bubblezone v2.0.0 pins charm.land v2 — use it (except with the canvas compositor).
8. list row stride = `delegate.Height()+delegate.Spacing()`.
9. Opening an overlay / changing state from a key handler should return a `tea.Cmd` (e.g. the
   textinput `Focus()`/blink cmd). Returning `nil` can leave the new frame unflushed under a PTY test
   harness (glyph) until the next event — return a command so bubbletea emits a follow-up frame.
