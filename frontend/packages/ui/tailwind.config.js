/** @type {import('tailwindcss').Config} */
export default {
  darkMode: "class",
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
    "../browser/src/**/*.{vue,js,ts,jsx,tsx}",
    "../quota/src/**/*.{vue,js,ts,jsx,tsx}",
    "../study/src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  corePlugins: {
    preflight: false,
  },
  theme: {
    extend: {
      colors: {
        primary: 'var(--token-primary)',
        'primary-hover': 'var(--token-primary-hover)',
        'primary-50': 'var(--token-primary-50)',
        'primary-100': 'var(--token-primary-100)',
        'primary-200': 'var(--token-primary-200)',
        'primary-300': 'var(--token-primary-300)',
        'primary-400': 'var(--token-primary-400)',
        'primary-500': 'var(--token-primary-500)',
        'primary-600': 'var(--token-primary-600)',
        'primary-700': 'var(--token-primary-700)',
        'primary-800': 'var(--token-primary-800)',
        'primary-900': 'var(--token-primary-900)',
        success: 'var(--token-success)',
        warning: 'var(--token-warning)',
        error: 'var(--token-error)',
        accent: 'var(--token-accent)',
        surface: 'var(--token-surface)',
        panel: 'var(--token-panel)',
      },
      fontFamily: {
        display: ["Bebas Neue", "system-ui", "sans-serif"],
        body: ["Manrope", "system-ui", "sans-serif"],
      },
      boxShadow: {
        glow: "0 12px 40px rgba(1, 139, 141, 0.35)",
      },
    },
  },
  plugins: [],
}

