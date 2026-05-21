export type ThemeVars = Record<string, string>;

export interface Theme {
  name: string;
  vars: ThemeVars;
}

export const THEMES: Record<string, Theme> = {
  "dark-indigo": { name: "Dark Indigo", vars: {
    "--bg-primary": "#0d0d1a", "--bg-secondary": "#13132b", "--bg-surface": "#1a1b3e",
    "--bg-elevated": "#222350", "--bg-hover": "#2a2b5e",
    "--border-primary": "#1e1f42", "--border-subtle": "#181935",
    "--text-primary": "#e2e8f0", "--text-secondary": "#94a3b8", "--text-muted": "#64748b",
    "--accent-blue": "#818cf8", "--accent-green": "#4ade80", "--accent-red": "#f87171", "--accent-yellow": "#fbbf24",
  }},
  "dark-purple": { name: "Dark Purple", vars: {
    "--bg-primary": "#120a1a", "--bg-secondary": "#1c112b", "--bg-surface": "#261a3e",
    "--bg-elevated": "#322550", "--bg-hover": "#3d2d5e",
    "--border-primary": "#2e2042", "--border-subtle": "#221835",
    "--text-primary": "#e8dff0", "--text-secondary": "#a894b8", "--text-muted": "#7a648b",
    "--accent-blue": "#a78bfa", "--accent-green": "#6ee7b7", "--accent-red": "#fca5a5", "--accent-yellow": "#fbbf24",
  }},
  "dark-green": { name: "Dark Green", vars: {
    "--bg-primary": "#0a140d", "--bg-secondary": "#102216", "--bg-surface": "#183021",
    "--bg-elevated": "#20402c", "--bg-hover": "#285037",
    "--border-primary": "#1e3a26", "--border-subtle": "#162c1d",
    "--text-primary": "#e0f0e4", "--text-secondary": "#94b89c", "--text-muted": "#648b6c",
    "--accent-blue": "#6ee7b7", "--accent-green": "#4ade80", "--accent-red": "#fca5a5", "--accent-yellow": "#fbbf24",
  }},
  "dark-amber": { name: "Dark Amber", vars: {
    "--bg-primary": "#14100a", "--bg-secondary": "#221c12", "--bg-surface": "#32281a",
    "--bg-elevated": "#403422", "--bg-hover": "#50422c",
    "--border-primary": "#3a3020", "--border-subtle": "#2c2418",
    "--text-primary": "#f0e8d8", "--text-secondary": "#b8a888", "--text-muted": "#8b7a5a",
    "--accent-blue": "#fbbf24", "--accent-green": "#a3e635", "--accent-red": "#fb923c", "--accent-yellow": "#f59e0b",
  }},
  "light": { name: "Light", vars: {
    "--bg-primary": "#f8fafc", "--bg-secondary": "#f1f5f9", "--bg-surface": "#e2e8f0",
    "--bg-elevated": "#cbd5e1", "--bg-hover": "#94a3b8",
    "--border-primary": "#cbd5e1", "--border-subtle": "#e2e8f0",
    "--text-primary": "#0f172a", "--text-secondary": "#334155", "--text-muted": "#64748b",
    "--accent-blue": "#6366f1", "--accent-green": "#16a34a", "--accent-red": "#dc2626", "--accent-yellow": "#eab308",
  }},
  "dark-dracula": { name: "Dracula Dark", vars: {
    "--bg-primary": "#282a36", "--bg-secondary": "#21222c", "--bg-surface": "#343746",
    "--bg-elevated": "#3d3f52", "--bg-hover": "#45475a",
    "--border-primary": "#3d3f52", "--border-subtle": "#343746",
    "--text-primary": "#f8f8f2", "--text-secondary": "#cdd6f4", "--text-muted": "#6c7086",
    "--accent-blue": "#bd93f9", "--accent-green": "#50fa7b", "--accent-red": "#ff5555", "--accent-yellow": "#f1fa8c",
  }},
  "dark-nord": { name: "Nord Dark", vars: {
    "--bg-primary": "#2e3440", "--bg-secondary": "#3b4252", "--bg-surface": "#434c5e",
    "--bg-elevated": "#4c566a", "--bg-hover": "#555e72",
    "--border-primary": "#4c566a", "--border-subtle": "#434c5e",
    "--text-primary": "#eceff4", "--text-secondary": "#d8dee9", "--text-muted": "#7a8699",
    "--accent-blue": "#88c0d0", "--accent-green": "#a3be8c", "--accent-red": "#bf616a", "--accent-yellow": "#ebcb8b",
  }},
  "dark-tokyo": { name: "Tokyo Night", vars: {
    "--bg-primary": "#1a1b26", "--bg-secondary": "#1f2135", "--bg-surface": "#24283b",
    "--bg-elevated": "#2f3348", "--bg-hover": "#3b3f58",
    "--border-primary": "#2f3348", "--border-subtle": "#24283b",
    "--text-primary": "#a9b1d6", "--text-secondary": "#9aa5ce", "--text-muted": "#565f89",
    "--accent-blue": "#7aa2f7", "--accent-green": "#73daca", "--accent-red": "#f7768e", "--accent-yellow": "#e0af68",
  }},
  "dark-catppuccin": { name: "Catppuccin Mocha", vars: {
    "--bg-primary": "#1e1e2e", "--bg-secondary": "#181825", "--bg-surface": "#313244",
    "--bg-elevated": "#45475a", "--bg-hover": "#515266",
    "--border-primary": "#45475a", "--border-subtle": "#313244",
    "--text-primary": "#cdd6f4", "--text-secondary": "#bac2de", "--text-muted": "#6c7086",
    "--accent-blue": "#89b4fa", "--accent-green": "#a6e3a1", "--accent-red": "#f38ba8", "--accent-yellow": "#f9e2af",
  }},
  // ─── Termux & macOS-inspired themes ───
  "termux": { name: "Termux Classic", vars: {
    "--bg-primary": "#000000", "--bg-secondary": "#050805", "--bg-surface": "#0a120a",
    "--bg-elevated": "#121e12", "--bg-hover": "#1c2e1c",
    "--border-primary": "#00ff6633", "--border-subtle": "#00ff6611",
    "--text-primary": "#00ff66", "--text-secondary": "#8affb5", "--text-muted": "#00aa44",
    "--accent-blue": "#00e5ff", "--accent-green": "#00ff66", "--accent-red": "#ff3366", "--accent-yellow": "#ffe600",
  }},
  "homebrew": { name: "Homebrew", vars: {
    "--bg-primary": "#000000", "--bg-secondary": "#040a05", "--bg-surface": "#08140a",
    "--bg-elevated": "#102412", "--bg-hover": "#18361b",
    "--border-primary": "#00ff3333", "--border-subtle": "#00ff3311",
    "--text-primary": "#00ff33", "--text-secondary": "#66ff88", "--text-muted": "#00aa22",
    "--accent-blue": "#3399ff", "--accent-green": "#00ff33", "--accent-red": "#ff3333", "--accent-yellow": "#ffff33",
  }},
  "red-sands": { name: "Red Sands", vars: {
    "--bg-primary": "#501b1a", "--bg-secondary": "#3e1514", "--bg-surface": "#2f100f",
    "--bg-elevated": "#642221", "--bg-hover": "#7a2b29",
    "--border-primary": "#ffcccc22", "--border-subtle": "#ffcccc0c",
    "--text-primary": "#ffd6d6", "--text-secondary": "#ffb3b3", "--text-muted": "#b07777",
    "--accent-blue": "#818cf8", "--accent-green": "#4ade80", "--accent-red": "#f87171", "--accent-yellow": "#fbbf24",
  }},
  "ocean-breeze": { name: "Ocean Breeze", vars: {
    "--bg-primary": "#041020", "--bg-secondary": "#020a16", "--bg-surface": "#071a33",
    "--bg-elevated": "#0c2647", "--bg-hover": "#12335c",
    "--border-primary": "#00e5ff33", "--border-subtle": "#00e5ff11",
    "--text-primary": "#e0f7fc", "--text-secondary": "#8ae2f2", "--text-muted": "#528c99",
    "--accent-blue": "#00e5ff", "--accent-green": "#4ade80", "--accent-red": "#ff6b6b", "--accent-yellow": "#facc15",
  }}
};

export const FONTS = [
  { value: "'Cascadia Code', 'Fira Code', 'Consolas', monospace", label: "Cascadia Code" },
  { value: "'Fira Code', 'Consolas', monospace", label: "Fira Code" },
  { value: "'JetBrains Mono', 'Consolas', monospace", label: "JetBrains Mono" },
  { value: "Consolas, 'Courier New', monospace", label: "Consolas" },
  { value: "'Source Code Pro', 'Consolas', monospace", label: "Source Code Pro" },
  { value: "'Iosevka', 'Consolas', monospace", label: "Iosevka" },
  { value: "'Monaspace Neon', 'Consolas', monospace", label: "Monaspace Neon" },
  { value: "'Geist Mono', 'Consolas', monospace", label: "Geist Mono" },
];

export function getStoredTheme(): string {
  try {
    return localStorage.getItem("contlib-theme") || "dark-indigo";
  } catch {
    return "dark-indigo";
  }
}

export function getStoredFont(): string {
  try {
    return localStorage.getItem("contlib-font") || "'Cascadia Code', 'Fira Code', 'Consolas', monospace";
  } catch {
    return "'Cascadia Code', 'Fira Code', 'Consolas', monospace";
  }
}

export function applyTheme(themeId: string) {
  const theme = THEMES[themeId] || THEMES["dark-indigo"];
  const root = document.documentElement;
  for (const [key, val] of Object.entries(theme.vars)) {
    root.style.setProperty(key, val);
  }
  try {
    localStorage.setItem("contlib-theme", themeId);
  } catch {}
}

export function applyFont(fontValue: string) {
  document.documentElement.style.setProperty("font-family", fontValue);
  try {
    localStorage.setItem("contlib-font", fontValue);
  } catch {}
}

export function getStoredFontSize(): number {
  try {
    return parseInt(localStorage.getItem("contlib-font-size") || "12", 10);
  } catch {
    return 12;
  }
}

export function applyFontSize(size: number) {
  const clamped = Math.max(9, Math.min(24, size));
  document.documentElement.style.setProperty("--font-size", `${clamped}px`);
  try {
    localStorage.setItem("contlib-font-size", String(clamped));
  } catch {}
}
