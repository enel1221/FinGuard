import { configureStore } from '@reduxjs/toolkit';
import { themeSlice } from './themeSlice';
import { sidebarSlice } from './sidebarSlice';

export const store = configureStore({
  reducer: {
    theme: themeSlice.reducer,
    sidebar: sidebarSlice.reducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
