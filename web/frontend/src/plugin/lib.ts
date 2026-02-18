import { Registry } from './registry';

export abstract class Plugin {
  abstract initialize(register: Registry): boolean | void;
}

export class Headlamp {
  static registerPlugin(pluginId: string, pluginObj: Plugin) {
    (window as any).plugins = (window as any).plugins || {};
    (window as any).plugins[pluginId] = pluginObj;
  }
}
