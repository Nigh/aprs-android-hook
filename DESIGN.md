---
name: APRS Tracks
description: Xianii dark interface for privately inspecting GPS journeys.
colors:
  primary: "oklch(81% 0.117 11.638)"
  primary-content: "oklch(26% 0 0)"
  accent: "oklch(80% 0.08 185)"
  accent-content: "oklch(26% 0 0)"
  base-100: "oklch(26% 0 0)"
  base-200: "oklch(20% 0 0)"
  base-300: "oklch(37% 0 0)"
  base-content: "oklch(96% 0.001 286.375)"
  text-muted: "oklch(96% 0.001 286.375 / 70%)"
  interactive-hover: "oklch(96% 0.001 286.375 / 8%)"
  interactive-active: "oklch(81% 0.117 11.638 / 16%)"
  focus: "oklch(81% 0.117 11.638)"
  error: "oklch(70% 0.18 25)"
  error-content: "oklch(20% 0.02 25)"
  success: "oklch(75% 0.10 150)"
  overlay: "oklch(0% 0 0 / 55%)"
typography:
  headline:
    fontFamily: '"Inter", "Noto Sans SC", ui-sans-serif, system-ui, sans-serif, "Apple Color Emoji", "Segoe UI Emoji"'
    fontSize: "1.5rem"
    fontWeight: 600
    lineHeight: 1.3
  title:
    fontSize: "1.125rem"
    fontWeight: 600
  body:
    fontFamily: '"Inter", "Noto Sans SC", ui-sans-serif, system-ui, sans-serif, "Apple Color Emoji", "Segoe UI Emoji"'
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontSize: "0.875rem"
    lineHeight: 1.5
  metadata:
    fontFamily: '"JetBrains Mono", "Fira Code", ui-monospace, "Cascadia Code", monospace'
    fontSize: "0.75rem"
rounded:
  selector: "0.5rem"
  field: "0.5rem"
  box: "0.5rem"
spacing:
  field-label: "0.5rem"
  field-help: "0.375rem"
  filter-gap: "1rem"
  mobile-filter-gap: "0.875rem"
  section: "1.5rem"
  shell: "2rem"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.primary-content}"
    rounded: "{rounded.field}"
    padding: "0 1rem"
    height: "3.125rem"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.base-content}"
    rounded: "{rounded.field}"
  button-error:
    backgroundColor: "{colors.error}"
    textColor: "{colors.error-content}"
    rounded: "{rounded.field}"
  input:
    backgroundColor: "{colors.base-100}"
    textColor: "{colors.base-content}"
    rounded: "{rounded.field}"
    height: "3.125rem"
  tab-selected:
    textColor: "{colors.primary}"
    padding: "0.75rem 0.25rem"
  track-selected:
    backgroundColor: "{colors.interactive-active}"
    padding: "0.625rem 1rem"
  workspace:
    backgroundColor: "{colors.base-100}"
    rounded: "{rounded.box}"
  badge-outline:
    backgroundColor: "transparent"
    textColor: "{colors.base-content}"
    rounded: "{rounded.selector}"
---

# Design System: APRS Tracks

## Overview

**Creative North Star: "Xianii private track workspace"**

The established [Nigh/xianii-theme](https://github.com/Nigh/xianii-theme) world is authoritative. The English-default interface, with Simplified Chinese available, uses `data-theme="xianii"` and the installed `@xianii/design-system/theme.css` entry, which combines the Tailwind and daisyUI adapters. These extracted values describe the current dark theme; upstream CSS custom properties remain the runtime source of truth.

The interface is compact and operational: dated journeys, measurements, and a dark map share one workspace. Implement directly in code and inspect the rendered result. The app mark is a geometric SVG; no generated raster assets or visual comp are required.

**Key Characteristics:**
- Dark neutral surfaces with restrained pink selection and near-white map points.
- Flat filters and inline totals above divided list and map regions.
- English and Simplified Chinese labels, readable measurements, and explicit interaction states.

## Colors

### Primary
- **Soft pink — primary / primary-content:** query and registration actions, selected tabs and track IDs, map line and end marker.
- **Pink wash — interactive-active:** selected track row surface.
- **Focus pink — focus:** visible keyboard outline; upstream `--color-focus` aliases `--color-primary`.

### Secondary
- **Soft teal — accent / accent-content:** start marker.
- **Semantic red — error / error-content:** request failures, deletion text, and confirmed destructive action.
- **Semantic green — success:** successful token changes.

### Neutral
- **Raised charcoal — base-100:** inputs, workspace, notices, and map popups.
- **Deep charcoal — base-200:** page background.
- **Divider gray — base-300:** borders and row separators.
- **Near white — base-content / text-muted:** text, supporting labels, and map point circles.
- **Light wash — interactive-hover:** hovered rows and tabs.
- **Dark veil — overlay:** native dialog backdrop.

**The Theme Authority Rule.** Use the existing public Xianii custom properties; do not create an independent palette or rename upstream tokens.

## Typography

Sans text follows `--font-sans`; hashes use `--font-mono`. The supplied stacks provide Chinese and system fallbacks; this app does not download fonts. Body uses `--font-size-body` and `--line-height-normal`; field labels and small supporting text use `--font-size-ui` / `--font-size-sm`. App headings use `--font-size-2xl`, while list and map headings use `--font-size-lg`; authentication headings use `--font-size-heading` (1.875rem). Token-page section headings use `--font-size-heading-small` (1.25rem). Semibold headings and medium buttons follow theme weights.

Summary totals use `--font-size-xl`, reducing to `--font-size-lg` on mobile. Measurements and tables use tabular numerals. Descriptive text and empty states use `--line-height-relaxed` (1.65).

## Layout

The app shell is capped at 1600px with centered margins and 2rem horizontal padding. A right-aligned native language select occupies a 4rem top toolbar on authentication and workspace screens; its width is 9rem. A compact header and underlined tabs precede the filter form and flat, wrapping summary. Desktop filters use six columns (`1fr 1.15fr 1.15fr 1fr 1.2fr auto`) with 1rem gaps. The bordered workspace pairs a 280–340px list with a flexible map; list rows scroll at 530px and the map height is `clamp(420px, 58vh, 640px)`.

At **1100px and below**, filters become three equal columns and summary gaps narrow. At **760px and below**, shell padding becomes 1rem, filters become two columns, summary guidance occupies its own line, and list then map stack vertically. The list scroll area is 280px and map height is 420px. Token creation stacks into one column; each token's note editor moves below its identity and delete action. The token page is capped at 1000px; authentication is centered within a 26rem panel.

## Elevation & Depth

Page regions are flat, separated by borders and tonal surfaces. Preserve daisyUI's existing control depth (`--depth: 1`, `--noise: 0`), including its subtle button/input shadows, rather than adding decorative card elevation. Native dialog depth comes from the theme overlay. MapLibre supplies map controls and popup structure; app popup surfaces, directional tips, and close-button interaction states are themed. Measurements use a label/value grid and at most two decimal places, matching the point detail table.

## Shapes

Selectors, fields, and containers use the upstream radius tokens, all currently gently rounded. Workspace overflow clips its internal list/map regions to the box radius. Borders use base-300. Endpoint markers are circles (28px) with a 2px base-100 border and localized start/end labels (S / E in English; 起 / 终 in Simplified Chinese). Selected lines use rounded joins and caps (4px); the overview uses 1.5px lines without points or endpoints. Selected map point circles use a 4px radius, increasing to 7px with a 2px dark stroke and pointer cursor on hover.

## Components

- **Buttons:** daisyUI primary, default, small ghost, and error variants; app buttons use theme medium weight. Disabled and busy states remain native. Keep the adapter's hover/press behavior.
- **Language:** English (`en`) is the default; Simplified Chinese (`zh-CN`) is available through the native labelled select. Options use each locale’s own name from the extensible registry in `web/src/lib/locales/index.js`. Keyed per-language JSON catalogs drive reactive copy, document title/description, `html lang`, aria labels, map endpoints, popup labels, and MapLibre controls. The `aprs-language` browser-local preference is independent of deployment timezone; dates still use the server’s timezone.
- **Fields:** real labels wrap native inputs and selects. Date inputs retain native date interaction; token input is password-masked. Field label/help spacing uses the upstream spacing properties.
- **Navigation:** tabs use a base-300 divider, muted resting labels, hover wash, and primary text plus a 2px selected underline. `aria-selected`, linked tab panels, roving tabindex, arrows, Home, and End express state and keyboard navigation.
- **Track rows:** compact full-width buttons with divided rows: ID and an ellipsized monospace device hash share the first line, followed by time range and wrapping measurements. The full device label is retained in the text and title. Selected rows use the pink wash and primary ID, with `aria-pressed` exposing selection. The newest track in the range is selected by default; clicking an overview line can select a track on another list page.
- **Containers:** the single list/map workspace and token rows organize content without decorative cards. Summary totals remain inline. The token-count badge is the existing outline badge.
- **Feedback:** alerts describe request errors; status messages identify loading/success and map failure. A failed map offers retry while list/statistics remain available. Point details use native `details` / `summary` and a scrolling table.
- **Focus and confirmation:** app `:focus-visible` uses a 3px theme-focus outline with 3px offset; adapter fields retain their own focus treatment. Token deletion uses a labelled native `dialog`, opened with `showModal()`, providing modal focus handling and Escape dismissal, with explicit cancel and error-colored confirmation.

## Do's and Don'ts

### Do:
- Do use the existing Xianii tokens and adapter components for every added surface.
- Do use keyed locale catalogs for English and Simplified Chinese labels and preserve the flat filter, summary, list/map hierarchy.
- Do keep selected state, keyboard focus, labels, and destructive confirmation explicit.
- Do inspect desktop, intermediate, and stacked mobile layouts after visual changes.

### Don't:
- Don't replace the established theme with a speculative visual world.
- Don't turn inline totals or divided rows into decorative cards.
- Don't hide map failure behind an unusable workspace.
