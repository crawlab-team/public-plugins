declare global {
  interface Window {
    _tp?: (pluginName: string, path: string) => string;
  }
}
