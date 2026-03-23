import type { Config } from "tailwindcss";

const config: Config = {
  // shadcn default: darkMode via class
  darkMode: ["class"],

  content: [
    "./src/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./app/**/*.{ts,tsx}",
  ],

  theme: {
    extend: {
      /* -----------------------------------------------------------------
         Typography — monospace everything, compact scale
         ----------------------------------------------------------------- */
      fontFamily: {
        // Override sans so all shadcn components (which use font-sans)
        // render in monospace without changing any component code
        sans: [
          "JetBrains Mono",
          "SF Mono",
          "Fira Code",
          "Cascadia Code",
          "ui-monospace",
          "monospace",
        ],
        mono: [
          "JetBrains Mono",
          "SF Mono",
          "Fira Code",
          "Cascadia Code",
          "ui-monospace",
          "monospace",
        ],
      },

      fontSize: {
        // Relay scale — denser than shadcn defaults
        xs:   ["11px", { lineHeight: "16px" }],
        sm:   ["12px", { lineHeight: "18px" }],
        base: ["13px", { lineHeight: "20px" }],
        lg:   ["14px", { lineHeight: "22px" }],
        xl:   ["16px", { lineHeight: "24px" }],
        // Rarely used — headings in modals/dialogs
        "2xl": ["18px", { lineHeight: "26px" }],
      },

      /* -----------------------------------------------------------------
         Spacing — tight rhythm for dev-tool density
         ----------------------------------------------------------------- */
      spacing: {
        "0.5": "2px",
        "1":   "4px",
        "1.5": "6px",
        "2":   "8px",
        "2.5": "10px",
        "3":   "12px",
        "4":   "16px",
        "5":   "20px",
        "6":   "24px",
        "8":   "32px",
      },

      /* -----------------------------------------------------------------
         Border radius — minimal radii, dev-tool feel
         ----------------------------------------------------------------- */
      borderRadius: {
        lg: "6px",   // cards, dialogs
        md: "4px",   // buttons, inputs, dropdowns
        sm: "2px",   // badges, tiny elements
      },

      /* -----------------------------------------------------------------
         Colors — Relay custom tokens accessible via Tailwind classes
         e.g. bg-relay-surface, text-relay-accent-blue
         ----------------------------------------------------------------- */
      colors: {
        relay: {
          base:    "#010409",
          surface: "#0d1117",
          raised:  "#161b22",
          overlay: "#21262d",

          accent: {
            blue:   "#58a6ff",
            green:  "#3fb950",
            amber:  "#d29922",
            red:    "#f85149",
            purple: "#bc8cff",
          },
        },
      },

      /* -----------------------------------------------------------------
         Shadows — none by default, only focus rings
         ----------------------------------------------------------------- */
      boxShadow: {
        // Minimal focus ring for accessibility
        "relay-focus": "0 0 0 2px rgba(88, 166, 255, 0.4)",
      },

      /* -----------------------------------------------------------------
         Keyframes & animations for subtle UI transitions
         ----------------------------------------------------------------- */
      keyframes: {
        "relay-fade-in": {
          from: { opacity: "0", transform: "translateY(2px)" },
          to:   { opacity: "1", transform: "translateY(0)" },
        },
        "relay-slide-in": {
          from: { opacity: "0", transform: "translateX(-4px)" },
          to:   { opacity: "1", transform: "translateX(0)" },
        },
      },
      animation: {
        "relay-fade-in":  "relay-fade-in 150ms ease-out",
        "relay-slide-in": "relay-slide-in 150ms ease-out",
      },

      /* -----------------------------------------------------------------
         Layout tokens
         ----------------------------------------------------------------- */
      width: {
        "sidebar": "232px",
      },
      minWidth: {
        "sidebar": "232px",
      },
      maxWidth: {
        "sidebar": "232px",
      },
    },
  },

  plugins: [
    // shadcn uses tailwindcss-animate
    require("tailwindcss-animate"),
  ],
};

export default config;
