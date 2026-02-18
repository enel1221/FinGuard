import { createSlice, PayloadAction } from '@reduxjs/toolkit';

export interface SidebarEntry {
  name: string;
  label: string;
  url: string;
  icon?: string;
  parent?: string;
  order?: number;
}

interface SidebarState {
  entries: SidebarEntry[];
}

const defaultEntries: SidebarEntry[] = [
  { name: 'dashboard', label: 'Dashboard', url: '/', icon: 'Dashboard', order: 0 },
  { name: 'projects', label: 'Projects', url: '/projects', icon: 'FolderSpecial', order: 5 },
  { name: 'cost-explorer', label: 'Cost Explorer', url: '/costs', icon: 'TrendingUp', order: 10 },
  { name: 'sources', label: 'Sources', url: '/sources', icon: 'Cloud', order: 20 },
  { name: 'budgets', label: 'Budgets', url: '/budgets', icon: 'AccountBalance', order: 30 },
  { name: 'recommendations', label: 'Recommendations', url: '/recommendations', icon: 'Lightbulb', order: 40 },
  { name: 'plugins', label: 'Plugins', url: '/plugins', icon: 'Extension', order: 90 },
  { name: 'settings', label: 'Settings', url: '/settings', icon: 'Settings', order: 100 },
];

const initialState: SidebarState = {
  entries: [...defaultEntries],
};

export const sidebarSlice = createSlice({
  name: 'sidebar',
  initialState,
  reducers: {
    addSidebarEntry(state, action: PayloadAction<SidebarEntry>) {
      state.entries = state.entries.filter(e => e.name !== action.payload.name);
      state.entries.push(action.payload);
      state.entries.sort((a, b) => (a.order ?? 50) - (b.order ?? 50));
    },
  },
});

export const { addSidebarEntry } = sidebarSlice.actions;
