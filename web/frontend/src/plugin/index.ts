import * as React from 'react';
import * as ReactDOM from 'react-dom';
import * as ReactRouter from 'react-router-dom';
import * as ReactRedux from 'react-redux';
import * as MuiMaterial from '@mui/material';
import * as Recharts from 'recharts';
import { Plugin, Headlamp } from './lib';
import { Registry } from './registry';
import {
  registerSidebarEntry,
  registerRoute,
  registerAppTheme,
  registerDashboardWidget,
  registerCostSourceView,
  registerPluginSettings,
} from './registry';
import { api } from '../lib/api';

(window as any).pluginLib = {
  React,
  ReactDOM,
  ReactRouter,
  ReactRedux,
  MuiMaterial,
  Recharts,
  ApiProxy: api,
  Plugin,
  Headlamp,
  registerSidebarEntry,
  registerRoute,
  registerAppTheme,
  registerDashboardWidget,
  registerCostSourceView,
  registerPluginSettings,
};

export async function fetchAndExecutePlugins(): Promise<void> {
  try {
    const resp = await fetch('/api/v1/frontend-plugins');
    if (!resp.ok) return;

    const plugins: { name: string; path: string }[] = await resp.json();

    for (const plugin of plugins) {
      try {
        const sourceResp = await fetch(`${plugin.path}/main.js`);
        if (!sourceResp.ok) continue;
        const source = await sourceResp.text();
        const executePlugin = new Function('pluginLib', source);
        executePlugin((window as any).pluginLib);
      } catch (e) {
        console.error(`Failed to load plugin ${plugin.name}:`, e);
      }
    }
  } catch {
    // Frontend plugins endpoint not available yet
  }
}

export async function initializePlugins(): Promise<void> {
  const plugins = (window as any).plugins || {};
  for (const pluginName of Object.keys(plugins)) {
    try {
      plugins[pluginName].initialize(new Registry());
    } catch (e) {
      console.error(`Plugin initialize() error in ${pluginName}:`, e);
    }
  }
}
