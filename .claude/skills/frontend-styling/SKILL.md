---
name: frontend-styling
description: Change CSS in the go-volunteer-media React frontend without causing a follow-up fix PR. Covers this app's global-cascade hazard (no CSS modules, 70+ unscoped .btn-primary/.btn-secondary rules across 12 page stylesheets), the inverted dark-mode neutral scale, the design tokens and which to use for CTAs, how to win a specificity fight correctly, and how to verify dark mode before opening a PR. Use whenever editing, adding, or reviewing any .css file under frontend/src.
---

# Frontend Styling

## Why this skill exists

Six of PRs #304–#320 were styling-only, and four of those existed solely to fix
the PR before them:

| PR | Fixed | What was missed |
|----|-------|-----------------|
| #305 | #304 | `Login.css`'s unscoped `.btn-secondary { width: 100% }` still won |
| #309 | #308 | `[data-theme='dark']` overrides used `--neutral-900/-800`, which invert |
| #313 | #312 | `UsersPage.css`'s `[data-theme='dark'] .btn-secondary` outranked the new rule |
| #314 | #312 | Used `--accent` (gold) for a CTA; the app's CTA color is `--brand` |

Every one is an instance of the same three facts below.

## Fact 1: every stylesheet is global

`frontend/src` has 51 `.css` files and **zero** `.module.css`. Each is a plain
`import './Foo.css'` that Vite concatenates into one bundle. There is no
component scoping. A rule written in `Login.css` applies to the Login page *and
to every other page in the app*.

Worse, `components/Modal.tsx` and `components/DateRangePicker.tsx` call
`createPortal(content, document.body)`. Their content is not a DOM descendant of
the page that rendered it, so any page-scoped ancestor selector you were relying
on does not match inside a modal.

Twelve files define unscoped `.btn-primary` / `.btn-secondary` rules — 72 of them:

```
pages/UsersPage.css            16    pages/AdminApiTokensPage.css     6
pages/Login.css                 7    pages/PhotoGallery.css           5
pages/BulkEditAnimalsPage.css   7    pages/Settings.css               4
pages/AdminAnimalTagsPage.css   7    pages/GroupPage.css              4
pages/Dashboard.css             6    pages/Form.css                   4
components/ConfirmDialog.css    3    components/admin/AnnouncementsTab.css 3
```

At equal specificity, the winner is decided by bundle load order. That is not
something you can reason about from the component tree, and it changes as files
are added.

**Rule: never style a button by reusing the global `.btn-primary` /
`.btn-secondary` class.** Give it a scoped class and style that:

```css
/* RequestCoverageRangeForm.css — the #305 fix */
.request-coverage-range-form__submit-btn { ... }
.request-coverage-range-form__cancel-btn { ... }
```

```tsx
<button className="request-coverage-range-form__submit-btn">Request Coverage</button>
```

Reach for the global class only for a throwaway internal screen where a
cascade surprise costs nothing.

## Fact 2: the neutral scale inverts in dark mode

In `index.css`, `[data-theme='dark']` does not merely darken — it **reverses**
the neutral ramp:

| Token | Light | Dark |
|-------|-------|------|
| `--neutral-100` | `#f3f4f6` | `#374151` |
| `--neutral-500` | `#6b7280` | `#d1d5db` |
| `--neutral-800` | `#1f2937` | `#f9fafb` |
| `--neutral-900` | `#111827` | `#ffffff` |

`--neutral-400` (`#9ca3af`) is the pivot and is the same in both.

So a base rule written with tokens is **already correct in dark mode**. Adding a
`[data-theme='dark']` block that swaps `--neutral-100` for `--neutral-800` does
not darken the element — it makes it near-white. That was the #308 bug, and the
#309 fix was to *delete* the override, not rewrite it.

**Rule: write base rules in tokens (`--surface`, `--bg`, `--text-primary`,
`--neutral-*`) and add no `[data-theme='dark']` block at all.** Add one only for
something tokens genuinely can't express (a shadow, an image treatment), and say
why in a comment.

## Fact 3: `--brand` is the CTA color, not `--accent`

```css
--brand:   #006b54;  /* HAWS green — primary CTA fill, light */
--brand:   #00a87e;  /*              dark */
--accent:  #F2B705;  /* gold — badges and highlights, NOT button fills */
```

Despite `--accent`'s comment saying "for CTAs", no button in the app fills with
it. #312 used gold for the Claim button and #314 changed it back to green.
Match the app, not the comment.

Shared badge tokens already exist — reuse them rather than re-deriving a tint:

```css
--warning-badge-bg:     color-mix(in srgb, var(--warning) 18%, transparent);
--warning-badge-border: color-mix(in srgb, var(--warning) 45%, transparent);
```

## Winning a specificity fight

When a global rule outranks yours, do **not** reach for `!important` and do not
rely on load order. Add a real ancestor class so your selector is strictly more
specific. The global offenders are two-class rules like
`[data-theme='dark'] .btn-secondary`, so three classes wins deterministically:

```css
/* NeedsCoverageList.css — the #313 fix */
.needs-coverage-list .needs-coverage-list__actions .needs-coverage-list__claim {
  background: var(--brand);
  color: var(--brand-contrast);
}
```

Leave a comment saying which global rule you are beating, as that file does.

## Before opening a styling PR

1. `cd frontend && npx tsc --noEmit`
2. `npx vitest run <the touched component>.test.tsx` — CSS changes don't break
   these, but class-name changes in TSX do.
3. **Look at it in dark mode.** Four of the six styling PRs in this window
   shipped a dark-mode bug. Run the app (`make dev-frontend`) and toggle the
   theme, or force `data-theme="dark"` on `<html>` in devtools. A green check on
   `vitest` proves nothing about color.
4. Check the mobile breakpoint (641px) — #308 exists because buttons truncated
   to "Mark optiona" on a phone.
5. Grep before you assume a class is yours:
   `grep -rn --include="*.css" "\.your-class" frontend/src`

## Adding a new component's styles

- One `Foo.css` next to `Foo.tsx`, every class prefixed `foo__`.
- Tokens only — no raw hex. `grep -rn "#[0-9a-fA-F]\{3,6\}" frontend/src/**/*.css`
  should not grow.
- No `[data-theme='dark']` block unless tokens can't do it.
- No unscoped element or utility selectors (`.card`, `button`, `.btn-*`) — they
  leak to all 51 stylesheets' worth of pages.
