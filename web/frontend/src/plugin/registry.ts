import { store } from '../redux/store';
import { addCustomAppTheme } from '../redux/themeSlice';
import { addSidebarEntry, type SidebarEntry } from '../redux/sidebarSlice';
import type { FinGuardTheme } from '../lib/AppTheme';

export function registerSidebarEntry(entry: SidebarEntry): void {
  store.dispatch(addSidebarEntry(entry));
}

export function registerAppTheme(theme: FinGuardTheme): void {
  store.dispatch(addCustomAppTheme(theme));
}

export interface RouteSpec {
  path: string;
  component: React.ComponentType;
  name: string;
  exact?: boolean;
}

export interface WidgetSpec {
  name: string;
  component: React.ComponentType;
  order?: number;
}

const registeredRoutes: RouteSpec[] = [];
const registeredWidgets: WidgetSpec[] = [];
const registeredCostSourceViews: Map<string, React.ComponentType> = new Map();
const registeredPluginSettings: Map<string, React.ComponentType> = new Map();

export function registerRoute(route: RouteSpec): void {
  registeredRoutes.push(route);
}

export function registerDashboardWidget(widget: WidgetSpec): void {
  registeredWidgets.push(widget);
  registeredWidgets.sort((a, b) => (a.order ?? 50) - (b.order ?? 50));
}

export function registerCostSourceView(type: string, component: React.ComponentType): void {
  registeredCostSourceViews.set(type, component);
}

export function registerPluginSettings(pluginId: string, component: React.ComponentType): void {
  registeredPluginSettings.set(pluginId, component);
}

export function getRegisteredRoutes(): RouteSpec[] {
  return [...registeredRoutes];
}

export function getRegisteredWidgets(): WidgetSpec[] {
  return [...registeredWidgets];
}

export function getCostSourceView(type: string): React.ComponentType | undefined {
  return registeredCostSourceViews.get(type);
}

export function getPluginSettingsComponent(pluginId: string): React.ComponentType | undefined {
  return registeredPluginSettings.get(pluginId);
}

export class Registry {
  registerSidebarEntry = registerSidebarEntry;
  registerRoute = registerRoute;
  registerAppTheme = registerAppTheme;
  registerDashboardWidget = registerDashboardWidget;
  registerCostSourceView = registerCostSourceView;
  registerPluginSettings = registerPluginSettings;
}
