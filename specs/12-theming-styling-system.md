# Theming and Styling System Specification

> Status: Draft
> Version: 2.0
> Last Updated: 2026-01-27

## Overview

This document specifies the theming and styling system for the Notes Editor application. The application maintains synchronized visual themes across React Web and Android platforms, providing both dark and light mode support with consistent color palettes, spacing, and typography.

**Implementation:** React (`clients/web/src/index.css`), Android (`clients/android/app/src/main/java/.../Theme.kt`)

**Key Principles:**
- **Platform-native implementation**: CSS custom properties for Web, Compose theme system for Android
- **Synchronized design tokens**: Matching colors, spacing, and visual language across platforms
- **Dark theme default**: Dark mode is the default experience on both platforms
- **Semantic naming**: Color and spacing variables named by purpose, not appearance

---

## Color Palette

### Dark Theme (Default)

| Token | Web CSS | Android Compose | Hex Value | Purpose |
|-------|---------|-----------------|-----------|---------|
| Background | `--bg` | `background` | `#0F1012` | Page/screen background |
| Panel | `--panel` | `panel` | `#15171A` | Card and panel backgrounds |
| Panel Border | `--panel-border` | `panelBorder` | `#2A2D33` | Borders around panels |
| Text | `--text` | `text` | `#E6E6E6` | Primary text color |
| Muted | `--muted` | `muted` | `#9AA0A6` | Secondary/dimmed text |
| Accent | `--accent` | `accent` | `#D9832B` | Primary accent (orange) |
| Accent Dim | `--accent-dim` | `accentDim` | `#7A4A1D` | Dimmed accent for backgrounds |
| Danger | `--danger` | `danger` | `#D66B6B` | Error and destructive actions |
| Input | `--input` | `input` | `#0F1114` | Input field backgrounds |
| Note | `--note` | `note` | `#101317` | Note view background |
| Button | N/A | `button` | `#1E2227` | Button background (Android) |
| Button Text | N/A | `buttonText` | `#E6E6E6` | Button text (Android) |
| Checkbox Fill | N/A | `checkboxFill` | `#E6E6E6` | Checkbox fill color (Android) |
| Shadow | `--shadow` | N/A | `rgba(0,0,0,0.35)` | Drop shadow color (Web) |

### Light Theme

| Token | Web CSS | Android Compose | Hex Value | Purpose |
|-------|---------|-----------------|-----------|---------|
| Background | `--bg` | `background` | `#E9F7F7` | Page/screen background |
| Panel | `--panel` | `panel` | `#F6FBFF` | Card and panel backgrounds |
| Panel Border | `--panel-border` | `panelBorder` | `#C7E3E6` | Borders around panels |
| Text | `--text` | `text` | `#1A2A2F` | Primary text color |
| Muted | `--muted` | `muted` | `#4F6F78` | Secondary/dimmed text |
| Accent | `--accent` | `accent` | `#3AA7A3` | Primary accent (teal) |
| Accent Dim | `--accent-dim` | `accentDim` | `#C9F1EF` | Dimmed accent for backgrounds |
| Danger | `--danger` | `danger` | `#D66B6B` | Error and destructive actions |
| Input | `--input` | `input` | `#F2FAFB` | Input field backgrounds |
| Note | `--note` | `note` | `#F9FDFF` | Note view background |
| Button | N/A | `button` | `#EEF6F8` | Button background (Android) |
| Button Text | N/A | `buttonText` | `#1A2A2F` | Button text (Android) |
| Checkbox Fill | N/A | `checkboxFill` | `#F2FAFB` | Checkbox fill color (Android) |
| Shadow | `--shadow` | N/A | `rgba(18,33,45,0.08)` | Drop shadow color (Web) |

### Background Gradients

**Dark Theme:**
```css
background: radial-gradient(circle at 20% 20%, #1a1c20 0%, #0f1012 55%);
```

**Light Theme:**
```css
background: radial-gradient(circle at 15% 15%, #f7fcff 0%, #def3f5 60%);
```

---

## Spacing System

### Web CSS Custom Properties

| Variable | Value | Usage |
|----------|-------|-------|
| `--space-1` | `6px` | Tight spacing, gaps between inline elements |
| `--space-2` | `10px` | Standard padding, form gaps |
| `--space-3` | `14px` | Panel padding, section margins |
| `--space-4` | `18px` | Large spacing between sections |
| `--radius` | `6px` | Border radius for panels and inputs |

### Android Compose Spacing

```kotlin
data class AppSpacing(
    val xs: Dp = 6.dp,   // Equivalent to --space-1
    val sm: Dp = 10.dp,  // Equivalent to --space-2
    val md: Dp = 14.dp,  // Equivalent to --space-3
    val lg: Dp = 18.dp,  // Equivalent to --space-4
    val xl: Dp = 24.dp,  // Extended spacing (Android only)
)
```

### Cross-Platform Spacing Mapping

| Purpose | Web | Android |
|---------|-----|---------|
| Extra small / tight | `--space-1` (6px) | `AppTheme.spacing.xs` (6.dp) |
| Small / standard | `--space-2` (10px) | `AppTheme.spacing.sm` (10.dp) |
| Medium / panel padding | `--space-3` (14px) | `AppTheme.spacing.md` (14.dp) |
| Large / section gaps | `--space-4` (18px) | `AppTheme.spacing.lg` (18.dp) |
| Extra large | N/A | `AppTheme.spacing.xl` (24.dp) |

---

## Typography

### Web Typography

**Font Stack:**
```css
--font: "IBM Plex Mono", "Menlo", "Consolas", "Liberation Mono", monospace;
```

**Base Styles:**
- Font size: `14px`
- Line height: `1.5`

**Heading Styles:**

| Element | Font Size | Font Weight | Additional |
|---------|-----------|-------------|------------|
| `h3` | `13px` | 600 | Uppercase, `letter-spacing: 0.04em`, muted color |
| `.heading.h1` | `16px` | 600 | - |
| `.heading.h2` | `14px` | 600 | Uppercase, `letter-spacing: 0.04em`, muted color |
| `.heading.h3` | `13px` | 600 | Accent color |
| `.heading.h4` | `12px` | 600 | Text color |

### Android Typography

**Font Family:**
```kotlin
val appFont = FontFamily(
    Font(R.font.jetbrains_mono_nerd_regular, weight = FontWeight.Normal),
    Font(R.font.jetbrains_mono_nerd_medium, weight = FontWeight.Medium),
    Font(R.font.jetbrains_mono_nerd_bold, weight = FontWeight.Bold),
)
```

**Text Styles:**

```kotlin
data class AppTypography(
    val body: TextStyle,      // 12sp, lineHeight 17sp
    val bodySmall: TextStyle, // 11sp, lineHeight 15sp
    val title: TextStyle,     // 16sp, lineHeight 20sp, Medium weight
    val section: TextStyle,   // 12sp, lineHeight 16sp, letterSpacing 0.4sp
    val label: TextStyle,     // 11sp, lineHeight 14sp
)
```

### Cross-Platform Typography Mapping

| Purpose | Web | Android |
|---------|-----|---------|
| Body text | `14px` | `body` (12sp) |
| Small text | `12px` | `bodySmall` (11sp) |
| Title / H1 | `16px` | `title` (16sp) |
| Section headers | `13px uppercase` | `section` (12sp) |
| Labels | `11-12px` | `label` (11sp) |

---

## CSS Custom Properties (Web)

### Root Variables Definition

```css
:root {
    /* Colors */
    --bg: #0f1012;
    --panel: #15171a;
    --panel-border: #2a2d33;
    --text: #e6e6e6;
    --muted: #9aa0a6;
    --accent: #d9832b;
    --accent-dim: #7a4a1d;
    --danger: #d66b6b;
    --input: #0f1114;
    --note: #101317;
    --shadow: rgba(0, 0, 0, 0.35);

    /* Spacing */
    --radius: 6px;
    --space-1: 6px;
    --space-2: 10px;
    --space-3: 14px;
    --space-4: 18px;

    /* Typography */
    --font: "IBM Plex Mono", "Menlo", "Consolas", "Liberation Mono", monospace;
}
```

### Light Theme Override

```css
body.theme-light {
    --bg: #e9f7f7;
    --panel: #f6fbff;
    --panel-border: #c7e3e6;
    --text: #1a2a2f;
    --muted: #4f6f78;
    --accent: #3aa7a3;
    --accent-dim: #c9f1ef;
    --danger: #d66b6b;
    --input: #f2fafb;
    --note: #f9fdff;
    --shadow: rgba(18, 33, 45, 0.08);
    background: radial-gradient(circle at 15% 15%, #f7fcff 0%, #def3f5 60%);
}
```

### Global Reset

```css
* {
    box-sizing: border-box;
}
```

---

## Compose Theme System (Android)

### Data Classes

```kotlin
data class AppColors(
    val background: Color,
    val panel: Color,
    val panelBorder: Color,
    val text: Color,
    val muted: Color,
    val accent: Color,
    val accentDim: Color,
    val danger: Color,
    val input: Color,
    val note: Color,
    val button: Color,
    val buttonText: Color,
    val checkboxFill: Color,
)

data class AppSpacing(
    val xs: Dp = 6.dp,
    val sm: Dp = 10.dp,
    val md: Dp = 14.dp,
    val lg: Dp = 18.dp,
    val xl: Dp = 24.dp,
)

data class AppTypography(
    val body: TextStyle,
    val bodySmall: TextStyle,
    val title: TextStyle,
    val section: TextStyle,
    val label: TextStyle,
)
```

### CompositionLocal Providers

```kotlin
private val LocalAppColors = staticCompositionLocalOf { /* default dark colors */ }
private val LocalAppSpacing = staticCompositionLocalOf { AppSpacing() }
private val LocalAppTypography = staticCompositionLocalOf { /* default typography */ }
```

### AppTheme Object

```kotlin
object AppTheme {
    val colors: AppColors
        @Composable get() = LocalAppColors.current
    val spacing: AppSpacing
        @Composable get() = LocalAppSpacing.current
    val typography: AppTypography
        @Composable get() = LocalAppTypography.current
}
```

### Theme Composable

```kotlin
@Composable
fun NotesEditorTheme(content: @Composable () -> Unit) {
    val colors = if (UserSettings.theme == "light") {
        // Light theme colors
    } else {
        // Dark theme colors (default)
    }
    val spacing = AppSpacing()
    val typography = LocalAppTypography.current

    CompositionLocalProvider(
        LocalAppColors provides colors,
        LocalAppSpacing provides spacing,
        LocalAppTypography provides typography,
    ) {
        content()
    }
}
```

### Usage in Composables

```kotlin
@Composable
fun MyComponent() {
    Box(
        modifier = Modifier
            .background(AppTheme.colors.panel)
            .padding(AppTheme.spacing.md)
    ) {
        Text(
            text = "Hello",
            color = AppTheme.colors.text,
            style = AppTheme.typography.body
        )
    }
}
```

---

## Component Styling

### Layout Components

#### Container
```css
.container {
    max-width: 840px;
    margin: 0 auto;
    padding: var(--space-3);
}
```

#### Topbar
```css
.topbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
}
```

#### Panel
```css
.panel {
    background: var(--panel);
    border: 1px solid var(--panel-border);
    border-radius: var(--radius);
    padding: var(--space-3);
    box-shadow: 0 12px 28px var(--shadow);
}
```

#### Brand
```css
.brand {
    display: flex;
    align-items: center;
    gap: var(--space-2);
}
```

### File Browser Components

#### File Tree
```css
.file-tree {
    display: flex;
    flex-direction: column;
    gap: 2px;
    max-height: 70vh;
    overflow-y: auto;
}
```

#### Tree Item
```css
.tree-item {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    padding: 4px 6px;
    border-radius: 4px;
    cursor: pointer;
    color: var(--muted);
}

.tree-item.file:hover,
.tree-item.dir:hover {
    background: #1c2026;
    color: var(--text);
}
```

#### Tree Toggle
```css
.tree-toggle {
    width: 18px;
    height: 18px;
    border-radius: 4px;
    border: 1px solid var(--panel-border);
    background: transparent;
    color: var(--muted);
    font-size: 12px;
    cursor: pointer;
}

.tree-toggle:hover {
    color: var(--accent);
    border-color: var(--accent-dim);
}
```

#### Tree Children
```css
.tree-children {
    margin-left: 14px;
    padding-left: var(--space-1);
    border-left: 1px dashed #24282e;
}
```

### Chat Components

#### Chat Panel
```css
.chat-panel {
    display: flex;
    flex-direction: column;
    height: calc(100vh - 100px);
}
```

#### Chat Messages
```css
.chat-messages {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-2) 0;
    min-height: 200px;
}
```

#### Chat Message
```css
.chat-message {
    padding: var(--space-2);
    border-radius: var(--radius);
    background: var(--input);
}

.chat-message.user {
    background: var(--accent-dim);
    margin-left: 20%;
}

.chat-message.assistant {
    margin-right: 20%;
}
```

#### Chat Form
```css
.chat-form {
    display: flex;
    gap: var(--space-2);
    margin-top: var(--space-2);
}

.chat-form textarea {
    flex: 1;
}

.chat-form button {
    align-self: flex-end;
}
```

### Noise Components

#### Noise Panel
```css
.noise-panel {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
}
```

#### Noise Shell
```css
.noise-shell {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    min-height: 520px;
}
```

#### Noise Button
```css
.noise-button {
    min-height: 110px;
    font-size: 20px;
    font-weight: 600;
    border-radius: 10px;
}
```

### Sleep Components

#### Sleep Form
```css
.sleep-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
}
```

#### Sleep Block
```css
.sleep-block {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.sleep-block input {
    background: #0f1114;
    border: 1px solid var(--panel-border);
    border-radius: var(--radius);
    color: var(--text);
    padding: 6px 8px;
    font-family: var(--font);
    font-size: 13px;
}
```

#### Sleep List
```css
.sleep-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 12px;
}
```

#### Sleep Line
```css
.sleep-line {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    padding: 4px 6px;
    border-radius: 4px;
    border: 1px solid var(--panel-border);
    background: #111418;
    color: var(--text);
}
```

### Note Components

#### Note View
```css
.note-view {
    border: 1px solid var(--panel-border);
    border-radius: var(--radius);
    background: var(--note);
    padding: var(--space-2);
    max-height: 420px;
    overflow-y: auto;
    font-size: 12px;
}
```

#### Note Line
```css
.note-line {
    white-space: pre-wrap;
    padding: 2px 4px;
}

.note-line.empty {
    min-height: 12px;
}
```

#### Note Heading
```css
.note-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    background: #151a1f;
    border-radius: 4px;
}

.note-heading .line-text {
    color: var(--accent);
}
```

#### Task Line
```css
.task-line {
    display: flex;
    align-items: center;
    gap: 6px;
}

.task-line input {
    accent-color: var(--accent);
    cursor: pointer;
}

.task-line.done .task-text {
    text-decoration: line-through;
    color: var(--muted);
}
```

#### Pin Form
```css
.pin-form {
    margin: 0;
}

.pin-action {
    border: 1px solid var(--accent-dim);
    background: transparent;
    color: var(--accent);
    padding: 2px 8px;
    border-radius: 999px;
    font-size: 11px;
    cursor: pointer;
}

.pin-action:hover {
    background: var(--accent-dim);
    color: var(--text);
}
```

---

## Button Variants

### Base Button
```css
.button {
    appearance: none;
    border: 1px solid var(--panel-border);
    background: #1e2227;
    color: var(--text);
    padding: 3px 10px;
    font-size: 12px;
    border-radius: var(--radius);
    cursor: pointer;
    transition: border-color 0.2s ease, color 0.2s ease, background 0.2s ease;
}

.button:hover {
    border-color: var(--accent);
    color: var(--accent);
}
```

### Ghost Button
```css
.button.ghost {
    background: transparent;
    color: var(--muted);
}

.button.ghost:hover {
    color: var(--text);
    border-color: var(--accent-dim);
}
```

### Ghost Danger Button
```css
.button.ghost.danger {
    color: var(--danger);
    border-color: #3a2323;
}

.button.ghost.danger:hover {
    color: #f2a3a3;
    border-color: var(--danger);
}
```

### Active Button
```css
.button.active {
    border-color: var(--accent);
    color: var(--accent);
}
```

### Button Variants Summary

| Variant | Background | Border | Text | Hover Effect |
|---------|------------|--------|------|--------------|
| Default | `#1e2227` | `--panel-border` | `--text` | Accent border and text |
| Ghost | Transparent | `--panel-border` | `--muted` | Text to `--text`, border to `--accent-dim` |
| Ghost Danger | Transparent | `#3a2323` | `--danger` | Lighter danger color, danger border |
| Active | Same | `--accent` | `--accent` | - |

---

## Form Styling

### Textarea
```css
textarea {
    width: 100%;
    background: var(--input);
    border: 1px solid var(--panel-border);
    border-radius: var(--radius);
    color: var(--text);
    padding: var(--space-2);
    font-family: var(--font);
    font-size: 13px;
    line-height: 1.5;
}

textarea:focus {
    outline: none;
    border-color: var(--accent);
}
```

### Form Row
```css
.form-row {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: var(--space-2);
    margin-bottom: var(--space-2);
}

.form-row.split {
    justify-content: space-between;
}
```

### Inline Form
```css
.inline-form {
    margin: 0;
}
```

### Checkbox
```css
input[type="checkbox"] {
    accent-color: var(--accent);
}
```

---

## Utility Classes

### Muted Text
```css
.muted {
    color: var(--muted);
    font-size: 12px;
}
```

### Mark (Status Indicator)
```css
mark {
    display: inline-block;
    margin-top: var(--space-1);
    background: #1f2328;
    color: var(--text);
    border: 1px solid var(--panel-border);
    padding: 3px 8px;
    border-radius: var(--radius);
    font-size: 12px;
}

mark.error {
    border-color: var(--danger);
    color: var(--danger);
}
```

### Horizontal Rule
```css
hr {
    border: none;
    border-top: 1px solid var(--panel-border);
    margin: var(--space-3) 0;
}
```

---

## Responsive Design

### Mobile Breakpoint

The application uses a single breakpoint at `600px` for mobile responsiveness.

```css
@media (max-width: 600px) {
    .container {
        padding: var(--space-2);
    }

    .panel {
        padding: var(--space-2);
    }

    .panel-header {
        flex-direction: column;
        align-items: flex-start;
    }

    .panel-header .actions {
        align-self: flex-end;
    }

    .sleep-row {
        flex-wrap: wrap;
    }
}
```

### Responsive Adjustments Summary

| Component | Desktop | Mobile (<600px) |
|-----------|---------|-----------------|
| Container padding | `--space-3` (14px) | `--space-2` (10px) |
| Panel padding | `--space-3` (14px) | `--space-2` (10px) |
| Panel header | Row layout | Column layout |
| Sleep row | No wrap | Wrap enabled |

---

## Theme Switching Mechanism

### Web Implementation

Theme switching is controlled by adding/removing the `theme-light` class on the `<body>` element.

**Dark Theme (Default):**
```html
<body>
    <!-- Uses :root variables -->
</body>
```

**Light Theme:**
```html
<body class="theme-light">
    <!-- Overrides :root variables -->
</body>
```

**JavaScript Theme Toggle Example:**
```javascript
function toggleTheme() {
    document.body.classList.toggle('theme-light');
}

function setTheme(theme) {
    if (theme === 'light') {
        document.body.classList.add('theme-light');
    } else {
        document.body.classList.remove('theme-light');
    }
}
```

### Android Implementation

Theme switching is controlled via `UserSettings.theme` and applied through the `NotesEditorTheme` composable.

**Theme Selection:**
```kotlin
@Composable
fun NotesEditorTheme(content: @Composable () -> Unit) {
    val colors = if (UserSettings.theme == "light") {
        // Light theme AppColors
    } else {
        // Dark theme AppColors (default)
    }
    // ...
}
```

**Theme Values:**
- `"light"` - Light theme
- `"dark"` or any other value - Dark theme (default)

---

## Cross-Platform Consistency

### Color Token Mapping

| Semantic Purpose | Web Variable | Android Property |
|-----------------|--------------|------------------|
| Page background | `--bg` | `AppTheme.colors.background` |
| Panel background | `--panel` | `AppTheme.colors.panel` |
| Panel border | `--panel-border` | `AppTheme.colors.panelBorder` |
| Primary text | `--text` | `AppTheme.colors.text` |
| Secondary text | `--muted` | `AppTheme.colors.muted` |
| Primary accent | `--accent` | `AppTheme.colors.accent` |
| Dimmed accent | `--accent-dim` | `AppTheme.colors.accentDim` |
| Error/danger | `--danger` | `AppTheme.colors.danger` |
| Input background | `--input` | `AppTheme.colors.input` |
| Note background | `--note` | `AppTheme.colors.note` |

### Spacing Token Mapping

| Size | Web Variable | Android Property |
|------|--------------|------------------|
| Extra Small | `--space-1` (6px) | `AppTheme.spacing.xs` (6.dp) |
| Small | `--space-2` (10px) | `AppTheme.spacing.sm` (10.dp) |
| Medium | `--space-3` (14px) | `AppTheme.spacing.md` (14.dp) |
| Large | `--space-4` (18px) | `AppTheme.spacing.lg` (18.dp) |
| Extra Large | N/A | `AppTheme.spacing.xl` (24.dp) |

### Typography Mapping

| Purpose | Web | Android |
|---------|-----|---------|
| Font family | IBM Plex Mono | JetBrains Mono Nerd |
| Body text | 14px | 12sp (`body`) |
| Small text | 12px | 11sp (`bodySmall`) |
| Titles | 16px | 16sp (`title`) |
| Section headers | 13px uppercase | 12sp (`section`) |
| Labels | 11-12px | 11sp (`label`) |

### Platform-Specific Extensions

**Android Only:**
- `AppColors.button` - Dedicated button background color
- `AppColors.buttonText` - Dedicated button text color
- `AppColors.checkboxFill` - Checkbox fill color
- `AppSpacing.xl` - Extra large spacing (24.dp)

**Web Only:**
- `--shadow` - Drop shadow RGBA color
- `--radius` - Border radius (6px)
- Background gradients via CSS

---

## Files

| Platform | File | Purpose |
|----------|------|---------|
| React Web | `clients/web/src/index.css` | All CSS styles and custom properties |
| Android | `clients/android/app/src/main/java/com/bartmann/noteseditor/Theme.kt` | Theme data classes and composables |
| Android | `clients/android/app/src/main/res/font/` | JetBrains Mono Nerd font files |

---

## Notes

### Design Token Philosophy

The theming system uses semantic naming for colors and spacing to ensure consistency and ease of maintenance:

- Colors are named by purpose (e.g., `accent`, `danger`) not appearance (e.g., `orange`, `red`)
- Spacing uses t-shirt sizing (`xs`, `sm`, `md`, `lg`, `xl`) for intuitive scaling
- Both platforms share the same design token values where possible

### CSS Variable Inheritance

On Web, all components inherit colors from CSS custom properties, allowing theme changes to cascade automatically when the `theme-light` class is toggled.

### Compose CompositionLocal

On Android, the `CompositionLocalProvider` pattern allows any composable in the tree to access theme values via `AppTheme.colors`, `AppTheme.spacing`, and `AppTheme.typography` without prop drilling.

### Font Differences

The platforms use different but visually similar monospace fonts:
- **Web**: IBM Plex Mono (with fallbacks to Menlo, Consolas, Liberation Mono)
- **Android**: JetBrains Mono Nerd (includes nerd font glyphs)

Both fonts provide the monospace aesthetic appropriate for a notes/code editing application.
