/**
 * FinGuard Plugin SDK
 *
 * Plugins are UMD bundles that execute in the browser context with access to
 * `window.pluginLib`. This SDK provides TypeScript types for that API.
 *
 * Usage:
 *   import { registerSidebarEntry, registerRoute, registerAppTheme, Plugin, Headlamp } from '@finguard/plugin-sdk';
 *
 *   class MyPlugin extends Plugin {
 *     initialize(register) {
 *       register.registerSidebarEntry({ name: 'my-page', label: 'My Page', url: '/my-page', icon: 'Extension', order: 50 });
 *       register.registerRoute({ path: '/my-page', component: MyPageComponent, name: 'my-page' });
 *     }
 *   }
 *   Headlamp.registerPlugin('my-plugin', new MyPlugin());
 */

export interface SidebarEntryProps {
  name: string;
  label: string;
  url: string;
  icon?: string;
  parent?: string;
  order?: number;
}

export interface RouteSpec {
  path: string;
  component: React.ComponentType;
  name: string;
  exact?: boolean;
}

export interface FinGuardTheme {
  name: string;
  base: 'light' | 'dark';
  primary?: string;
  secondary?: string;
  accent?: string;
  text?: { primary?: string };
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

export interface WidgetSpec {
  name: string;
  component: React.ComponentType;
  order?: number;
}

export interface PluginRegistry {
  registerSidebarEntry(entry: SidebarEntryProps): void;
  registerRoute(route: RouteSpec): void;
  registerAppTheme(theme: FinGuardTheme): void;
  registerDashboardWidget(widget: WidgetSpec): void;
  registerCostSourceView(type: string, component: React.ComponentType): void;
  registerPluginSettings(pluginId: string, component: React.ComponentType): void;
}

export abstract class Plugin {
  abstract initialize(register: PluginRegistry): boolean | void;
}

export class Headlamp {
  static registerPlugin(pluginId: string, pluginObj: Plugin): void {
    const win = window as any;
    win.plugins = win.plugins || {};
    win.plugins[pluginId] = pluginObj;
  }
}

export const ApiProxy = (window as any)?.pluginLib?.ApiProxy;
