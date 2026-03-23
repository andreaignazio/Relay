# Relay Theme — shadcn/ui Integration

Dev-tool aesthetic dark theme for the Relay chat application.
Built on top of shadcn/ui with minimal overrides.

## Files

```
relay-theme/
├── globals.css              # CSS variables + utility classes (replace shadcn's globals.css)
├── tailwind.config.ts       # Font, scale, spacing, colors (replace your tailwind.config)
├── relay-tokens.ts          # TypeScript design tokens (copy to src/lib/theme/)
├── component-overrides.css  # Documentation: what to tweak in each shadcn component
└── README.md
```

## Setup

### 1. Install shadcn/ui

```bash
npx shadcn@latest init
```

When prompted:
- Style: **Default**
- Base color: **Slate** (we override everything anyway)
- CSS variables: **Yes**

### 2. Replace globals.css

Copy `globals.css` to your project, replacing the one shadcn generated:

```bash
cp relay-theme/globals.css src/app/globals.css
# or wherever your global styles live
```

### 3. Replace tailwind.config.ts

```bash
cp relay-theme/tailwind.config.ts tailwind.config.ts
```

Merge with your existing config if you have custom content paths.

### 4. Copy design tokens

```bash
cp relay-theme/relay-tokens.ts src/lib/theme/tokens.ts
```

### 5. Install JetBrains Mono

The font is loaded via Google Fonts in globals.css. For self-hosting:

```bash
npm install @fontsource/jetbrains-mono
```

Then in your layout:
```ts
import '@fontsource/jetbrains-mono/400.css';
import '@fontsource/jetbrains-mono/500.css';
```

And remove the `@import url(...)` line from globals.css.

### 6. Install shadcn components

```bash
npx shadcn@latest add button input textarea dropdown-menu dialog \
  tooltip command scroll-area avatar badge separator popover \
  context-menu skeleton
```

### 7. Apply component tweaks

Open `component-overrides.css` and follow the instructions for each component.
The changes are small — mostly reducing heights and padding.

## Usage

### CSS variables (in JSX/CSS)

```css
background-color: hsl(var(--background));
color: hsl(var(--foreground));
border-color: hsl(var(--border));
```

### Relay custom tokens (in JSX/CSS)

```css
background: var(--relay-bg-surface);
color: var(--relay-text-muted);
border-color: var(--relay-border-subtle);
```

### Tailwind classes

```tsx
<div className="bg-relay-surface text-relay-accent-blue rounded-md" />
```

### TypeScript tokens

```ts
import { relayTheme, getAvatarColor } from '@/lib/theme/tokens';

// Dynamic styles
const style = { color: relayTheme.accent.blue };

// Avatar color assignment
const { bg, text } = getAvatarColor(user.id);
```

## Design principles

1. **Monospace everywhere** — JetBrains Mono is the only font
2. **Dense but readable** — 13px base, 4-16px spacing scale
3. **Dark-only** — no light mode toggle (for now)
4. **Functional color** — accents mean something (blue=interactive, green=online, red=urgent)
5. **Minimal radius** — 4px default, 6px for large containers
6. **Two weights** — 400 regular, 500 medium. Nothing heavier.
