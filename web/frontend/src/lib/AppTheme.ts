export interface FinGuardTheme {
  name: string;
  base: 'light' | 'dark';
  primary?: string;
  secondary?: string;
  accent?: string;
  text?: { primary?: string };
  link?: { color?: string };
  background?: { default?: string; surface?: string; muted?: string };
  sidebar?: {
    background?: string;
    color?: string;
    selectedBackground?: string;
    selectedColor?: string;
    actionBackground?: string;
  };
  navbar?: { background?: string; color?: string };
  radius?: number;
  buttonTextTransform?: 'uppercase' | 'none';
  fontFamily?: string[];
}

export const defaultDarkTheme: FinGuardTheme = {
  name: 'FinGuard Dark',
  base: 'dark',
  primary: '#22c55e',
  secondary: '#6366f1',
  accent: '#22c55e',
  background: { default: '#0f1117', surface: '#1a1d27', muted: '#232733' },
  sidebar: {
    background: '#1a1d27',
    color: '#8b8fa3',
    selectedBackground: '#232733',
    selectedColor: '#22c55e',
    actionBackground: '#22c55e',
  },
  navbar: { background: '#1a1d27', color: '#e4e6ed' },
  radius: 8,
  buttonTextTransform: 'none',
  fontFamily: ['-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'sans-serif'],
};

export const defaultLightTheme: FinGuardTheme = {
  name: 'FinGuard Light',
  base: 'light',
  primary: '#16a34a',
  secondary: '#6366f1',
  accent: '#16a34a',
  background: { default: '#f8fafc', surface: '#ffffff', muted: '#f1f5f9' },
  sidebar: {
    background: '#ffffff',
    color: '#64748b',
    selectedBackground: '#f0fdf4',
    selectedColor: '#16a34a',
    actionBackground: '#16a34a',
  },
  navbar: { background: '#ffffff', color: '#1e293b' },
  radius: 8,
  buttonTextTransform: 'none',
};

export const defaultThemes: FinGuardTheme[] = [defaultDarkTheme, defaultLightTheme];
