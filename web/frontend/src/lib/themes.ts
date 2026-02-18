import { createTheme, Theme } from '@mui/material/styles';
import type { FinGuardTheme } from './AppTheme';

export function createMuiTheme(appTheme: FinGuardTheme): Theme {
  const isDark = appTheme.base === 'dark';

  return createTheme({
    palette: {
      mode: isDark ? 'dark' : 'light',
      primary: { main: appTheme.primary || (isDark ? '#22c55e' : '#16a34a') },
      secondary: { main: appTheme.secondary || '#6366f1' },
      background: {
        default: appTheme.background?.default || (isDark ? '#0f1117' : '#f8fafc'),
        paper: appTheme.background?.surface || (isDark ? '#1a1d27' : '#ffffff'),
      },
      text: {
        primary: appTheme.text?.primary || (isDark ? '#e4e6ed' : '#1e293b'),
      },
    },
    shape: {
      borderRadius: appTheme.radius ?? 8,
    },
    typography: {
      fontFamily: appTheme.fontFamily?.join(', ') || '-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
      button: {
        textTransform: appTheme.buttonTextTransform || 'none',
      },
    },
    components: {
      MuiButton: {
        styleOverrides: {
          root: { borderRadius: appTheme.radius ?? 8 },
        },
      },
      MuiCard: {
        styleOverrides: {
          root: {
            backgroundColor: appTheme.background?.surface || (isDark ? '#1a1d27' : '#ffffff'),
            borderRadius: appTheme.radius ?? 8,
          },
        },
      },
      MuiPaper: {
        styleOverrides: {
          root: { backgroundImage: 'none' },
        },
      },
    },
  });
}
