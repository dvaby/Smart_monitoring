export interface AppState {
  adminConfigured: boolean;
  authenticated: boolean;
  telegramConfigured: boolean;
  proxyConfigured: boolean;
  chatId: string;
}

interface WailsApp {
  GetState(): Promise<AppState>;
  CreateAdmin(username: string, password: string): Promise<void>;
  Login(username: string, password: string): Promise<void>;
  Logout(): Promise<void>;
  SaveTelegramSettings(token: string, chatId: string, proxyUrl: string): Promise<void>;
  TestTelegram(token: string, chatId: string, proxyUrl: string): Promise<string>;
  HideWindow(): Promise<void>;
}

declare global {
  interface Window {
    go?: { wailsbind?: { Bindings?: WailsApp } };
  }
}

export function appBridge(): WailsApp {
  const app = window.go?.wailsbind?.Bindings;
  if (!app) throw new Error('Bridge Wails tidak tersedia.');
  return app;
}
