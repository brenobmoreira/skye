import type { Mode } from './mode';

export interface ControlsApi {
  Hide(): Promise<void>;
  ToggleMaximise(): Promise<void>;
  Quit(): Promise<void>;
}

export interface ControlsRuntime {
  WindowMinimise(): void;
  WindowToggleMaximise(): void;
  WindowIsMaximised?(): Promise<boolean>;
  Quit(): void;
}

export function windowControls(mode: Mode, api: () => ControlsApi, runtime: () => ControlsRuntime) {
  const windowsApp = mode === 'windows-app';
  return {
    minimise: () => runtime().WindowMinimise(),
    toggleMaximise: () => (windowsApp ? runtime().WindowToggleMaximise() : void api().ToggleMaximise()),
    close: () => (windowsApp ? runtime().Quit() : void api().Hide()),
    quitAll: async () => {
      if (!windowsApp) return api().Quit();
      try {
        await api().Quit();
      } catch {}
      runtime().Quit();
    },
    isMaximised: async () => (mode === 'browser' ? false : ((await runtime().WindowIsMaximised?.()) ?? false)),
  };
}
