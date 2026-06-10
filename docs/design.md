# Design

## Visual identity

The operator panel is an internal tool, not a marketing surface. The tone is
operational, sober and information-dense: a clean, neutral interface that lets an
operator scan many ArgoCD apps at a glance, with soft rounded corners and a
single blue accent. It supports light and dark mode automatically, following the
user's `prefers-color-scheme`. The whole UI lives in a single embedded
`cmd/server/index.html` served at the site root.

## Colors

Colors are defined as CSS custom properties on `:root`, with a dark-mode override
inside `@media (prefers-color-scheme: dark)`. Use these tokens; do not hardcode
hex values in markup.

- Surfaces: `--bg` (page), `--surface` (cards/controls), `--surface-2` (insets),
  `--border`.
- Text: `--text` (primary), `--text-soft` (secondary), `--text-faint` (muted).
- Accent: `--accent` (blue `#2f6feb` in light mode) and `--accent-soft` for tints;
  used for primary buttons, links and focus rings.
- Semantic pairs, each a foreground + a soft background, used for status badges
  and messages: `--green`/`--green-bg`, `--amber`/`--amber-bg`,
  `--red`/`--red-bg`, `--blue`/`--blue-bg`, `--gray`/`--gray-bg`,
  `--indigo`/`--indigo-bg`.
- `--shadow` for the standard card/control elevation.

## Typography

- Body font is the system UI stack: `-apple-system, BlinkMacSystemFont,
  "Segoe UI", Roboto, Helvetica, Arial, sans-serif`.
- Monospace (branches, tokens, IDs) uses `ui-monospace, SFMono-Regular,
  "SF Mono", Menlo, Consolas, monospace` via the `.mono` class.
- Base size is 14px with line-height 1.5. Section headings are small,
  uppercase, letter-spaced and use `--text-soft`. Numeric values use
  `font-variant-numeric: tabular-nums` for alignment.

## Spacing & layout

- Content is centered in a `max-width: 1100px` container with 20px padding.
- Spacing uses small, consistent steps (≈ 4 / 6 / 8 / 10 / 14 / 18 / 20px gaps
  and paddings); cards sit in a CSS grid with a 10px gap.
- Corner radius uses the `--radius` token (12px) for cards and 8–10px for inputs
  and buttons; badges are fully pill-shaped (999px).
- Single responsive breakpoint at `max-width: 560px`, where card actions go
  full-width and the card header wraps.
- A sticky, translucent (`backdrop-filter` blur) header holds the brand,
  a refresh-interval combobox (`#refreshInterval`), an auto-refresh toggle
  (`#autorefresh`), a manual refresh button, and a token button.

## Shared components

All styles and components live inline in `cmd/server/index.html` (there is no
separate CSS/JS bundle). Reuse the existing component classes instead of creating
parallel styles:

- `.btn` (with `.primary`, `.ghost`, `.danger`, `.small` modifiers) for actions.
- `.card` / `.card-top` / `.card-meta` for the per-app rows, grouped by
  environment under `.group` / `.group-head`.
- `.badge` (with semantic color modifiers) for status chips.
- `.overlay` + `.modal` for dialogs (lock, token), `.field` for form rows.
- `.toast` (with `.ok` / `.err` / `.info`) for transient notifications.

The color tokens are the theme: extend the palette by adding `--*` variables in
both the `:root` and the dark-mode block, never by inlining colors.

## UI states

- **Loading**: a `.loading` block with a `.spin` spinner and "Cargando
  aplicaciones…", shown only on the first load when no apps are cached.
- **Empty**: a dashed `.empty` block — "No hay aplicaciones" when the agent
  returns nothing, or "Sin resultados" when filters match nothing.
- **Error**: load failures render an `.empty` block ("No se pudo cargar" /
  "Error de red"); per-app sync errors render an inline `.statusmsg`; transient
  action failures surface as an `.err` toast.

## Accessibility & i18n

- The panel UI is in Spanish (`<html lang="es">`); copy, placeholders and toasts
  are Spanish. There is no i18n framework — it is single-locale.
- Controls use real `<button>`, `<label>` and `<input>` elements with `title`
  hints; inputs show a visible focus state via `--accent`. Color is paired with
  text labels (badges carry a word, not just a color) so status is not conveyed
  by color alone.
