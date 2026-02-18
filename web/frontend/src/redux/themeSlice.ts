import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import type { FinGuardTheme } from '../lib/AppTheme';
import { defaultThemes, defaultDarkTheme } from '../lib/AppTheme';

interface ThemeState {
  name: string;
  appThemes: FinGuardTheme[];
}

function getInitialThemeName(): string {
  const stored = localStorage.getItem('finguardThemePreference');
  if (stored) return stored;
  if (window.matchMedia?.('(prefers-color-scheme: light)').matches) return 'FinGuard Light';
  return defaultDarkTheme.name;
}

const initialState: ThemeState = {
  name: getInitialThemeName(),
  appThemes: [...defaultThemes],
};

export const themeSlice = createSlice({
  name: 'theme',
  initialState,
  reducers: {
    setTheme(state, action: PayloadAction<string>) {
      state.name = action.payload;
      localStorage.setItem('finguardThemePreference', action.payload);
    },
    addCustomAppTheme(state, action: PayloadAction<FinGuardTheme>) {
      state.appThemes = state.appThemes.filter(t => t.name !== action.payload.name);
      state.appThemes.push(action.payload);
    },
  },
});

export const { setTheme, addCustomAppTheme } = themeSlice.actions;
