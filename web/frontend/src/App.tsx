import { useEffect, useMemo } from 'react';
import { Provider, useSelector } from 'react-redux';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ThemeProvider, CssBaseline, Box } from '@mui/material';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { store, type RootState } from './redux/store';
import { createMuiTheme } from './lib/themes';
import { defaultDarkTheme } from './lib/AppTheme';
import Sidebar from './components/Sidebar/Sidebar';
import Dashboard from './components/Dashboard/Dashboard';
import CostExplorer from './components/CostExplorer/CostExplorer';
import Sources from './components/Sources/Sources';
import Budgets from './components/Budgets/Budgets';
import Recommendations from './components/Recommendations/Recommendations';
import PluginsPage from './components/App/Plugins';
import SettingsPage from './components/Settings/Settings';
import ProjectManager from './components/App/ProjectManager';
import ProjectDetail from './components/Project/ProjectDetail';
import UserAccount from './components/App/UserAccount';
import LoginPage from './components/App/Login';
import { fetchAndExecutePlugins, initializePlugins } from './plugin';
import { getRegisteredRoutes } from './plugin/registry';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, staleTime: 30_000 } },
});

function AppContent() {
  const themeName = useSelector((state: RootState) => state.theme.name);
  const themes = useSelector((state: RootState) => state.theme.appThemes);

  const currentTheme = useMemo(() => {
    return themes.find(t => t.name === themeName) ?? defaultDarkTheme;
  }, [themeName, themes]);

  const muiTheme = useMemo(() => createMuiTheme(currentTheme), [currentTheme]);
  const pluginRoutes = getRegisteredRoutes();

  useEffect(() => {
    fetchAndExecutePlugins().then(initializePlugins);
  }, []);

  return (
    <ThemeProvider theme={muiTheme}>
      <CssBaseline />
      <BrowserRouter>
        <Box sx={{ display: 'flex', minHeight: '100vh' }}>
          <Sidebar />
          <Box component="main" sx={{ flex: 1, p: 3, overflow: 'auto' }}>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/costs" element={<CostExplorer />} />
              <Route path="/sources" element={<Sources />} />
              <Route path="/budgets" element={<Budgets />} />
              <Route path="/recommendations" element={<Recommendations />} />
              <Route path="/plugins" element={<PluginsPage />} />
              <Route path="/settings" element={<SettingsPage />} />
              <Route path="/projects" element={<ProjectManager />} />
              <Route path="/projects/:projectId" element={<ProjectDetail />} />
              <Route path="/account" element={<UserAccount />} />
              <Route path="/auth/login" element={<LoginPage />} />
              {pluginRoutes.map(route => (
                <Route key={route.path} path={route.path} element={<route.component />} />
              ))}
            </Routes>
          </Box>
        </Box>
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default function App() {
  return (
    <Provider store={store}>
      <QueryClientProvider client={queryClient}>
        <AppContent />
      </QueryClientProvider>
    </Provider>
  );
}
