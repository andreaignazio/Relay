/**
 * Relay — Design Tokens
 * 
 * Centralized constants for colors, typography, spacing, and component tokens.
 * Use these when you need design values in TypeScript (e.g. dynamic styles,
 * canvas rendering, chart colors, avatar color assignment).
 * 
 * For CSS/Tailwind usage, prefer the CSS variables in globals.css or the
 * Tailwind classes in tailwind.config.ts.
 */

// =============================================================================
// Backgrounds
// =============================================================================

export const bg = {
  base:    "#010409",  // deepest layer (app chrome, sidebar)
  surface: "#0d1117",  // main content panels
  raised:  "#161b22",  // cards, inputs, elevated elements
  overlay: "#21262d",  // dropdowns, tooltips, modals
} as const;

// =============================================================================
// Text
// =============================================================================

export const text = {
  primary:   "#e6edf3",  // headings, active items, message body
  secondary: "#c9d1d9",  // message body (relaxed), descriptions
  muted:     "#8b949e",  // timestamps, metadata, helper text
  faint:     "#7d8590",  // placeholders, disabled labels
  ghost:     "#484f58",  // section labels, divider text, hints
} as const;

// =============================================================================
// Accents
// =============================================================================

export const accent = {
  blue:   "#58a6ff",  // links, focus rings, primary actions, mentions
  green:  "#3fb950",  // online status, success, confirmations
  amber:  "#d29922",  // warnings, away status, thread indicators
  red:    "#f85149",  // errors, destructive, unread badges
  purple: "#bc8cff",  // secondary accent, highlights, tags
} as const;

// =============================================================================
// Borders
// =============================================================================

export const border = {
  subtle:  "rgba(110, 118, 129, 0.15)",  // panel dividers, code blocks
  default: "rgba(110, 118, 129, 0.30)",  // input borders, cards
  strong:  "rgba(110, 118, 129, 0.50)",  // active borders, emphasis
  solid:   "#30363d",                     // when you need a non-alpha border
} as const;

// =============================================================================
// Typography
// =============================================================================

export const font = {
  family: "'JetBrains Mono', 'SF Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace",
  
  size: {
    xs:   "11px",  // section labels, keyboard shortcuts
    sm:   "12px",  // timestamps, metadata, sidebar items
    base: "13px",  // message body, input text, default
    lg:   "14px",  // channel headers, dialog titles
    xl:   "16px",  // page titles (rare)
  },

  lineHeight: {
    xs:   "16px",
    sm:   "18px",
    base: "20px",
    lg:   "22px",
    xl:   "24px",
  },

  weight: {
    regular: 400,
    medium:  500,
  },
} as const;

// =============================================================================
// Spacing
// =============================================================================

export const spacing = {
  /** 4px — inline gap between icon and label */
  xs:  4,
  /** 8px — component internal padding */
  sm:  8,
  /** 12px — list item vertical padding, compact gaps */
  md:  12,
  /** 16px — section gaps, card padding */
  lg:  16,
  /** 20px — main content padding */
  xl:  20,
  /** 24px — large section separators */
  xxl: 24,
} as const;

// =============================================================================
// Radii
// =============================================================================

export const radius = {
  sm: 2,   // badges, tiny pills
  md: 4,   // buttons, inputs, dropdowns, avatars
  lg: 6,   // cards, dialogs, panels
} as const;

// =============================================================================
// Layout
// =============================================================================

export const layout = {
  sidebarWidth: 232,
  messageAvatarSize: 28,
  headerHeight: 48,
  composerMinHeight: 44,
} as const;

// =============================================================================
// Avatar colors — cycle through these for user avatar backgrounds
// =============================================================================

export const avatarColors = [
  { bg: "rgba(88, 166, 255, 0.12)",  text: accent.blue },
  { bg: "rgba(63, 185, 80, 0.12)",   text: accent.green },
  { bg: "rgba(188, 140, 255, 0.12)", text: accent.purple },
  { bg: "rgba(210, 153, 34, 0.12)",  text: accent.amber },
  { bg: "rgba(248, 81, 73, 0.12)",   text: accent.red },
] as const;

/**
 * Deterministic avatar color based on user ID.
 * Same user always gets the same color.
 */
export function getAvatarColor(userId: string) {
  let hash = 0;
  for (let i = 0; i < userId.length; i++) {
    hash = ((hash << 5) - hash + userId.charCodeAt(i)) | 0;
  }
  return avatarColors[Math.abs(hash) % avatarColors.length];
}

// =============================================================================
// Syntax highlighting — token colors for code blocks
// =============================================================================

export const syntax = {
  keyword:  "#ff7b72",  // func, if, return, var, const
  function: "#d2a8ff",  // function names, method calls
  string:   "#a5d6ff",  // string literals
  number:   "#79c0ff",  // numeric literals, types
  type:     "#79c0ff",  // type names
  comment:  "#484f58",  // comments
  param:    "#ffa657",  // parameters, properties
  operator: "#ff7b72",  // =, ==, =>, +, -
  punct:    "#c9d1d9",  // brackets, semicolons
} as const;

// =============================================================================
// Re-export everything as a single theme object
// =============================================================================

export const relayTheme = {
  bg,
  text,
  accent,
  border,
  font,
  spacing,
  radius,
  layout,
  avatarColors,
  getAvatarColor,
  syntax,
} as const;

export default relayTheme;
