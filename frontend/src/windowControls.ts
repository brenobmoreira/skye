import type { Mode } from './mode';

export interface ControlsApi {
  Hide(): Promise<void>;
  ToggleMaximise(): Promise<void>;
  Quit(): Promise<void>;
  SetFocused(focused: boolean): Promise<void>;
}

export interface ControlsRuntime {
  WindowMinimise(): void;
  WindowToggleMaximise(): void;
  WindowIsMaximised?(): Promise<boolean>;
  Quit(): void;
}

const UNFOCUS_TIMEOUT = 300;

const delay = (ms: number) => new Promise<void>((resolve) => { setTimeout(resolve, ms); });

export function windowControls(
  mode: Mode,
  api: () => ControlsApi,
  runtime: () => ControlsRuntime,
  ready: () => boolean = () => true,
  wait: (ms: number) => Promise<void> = delay,
) {
  const windowsApp = mode === 'windows-app';
  const closeApp = async () => {
    if (ready()) await Promise.race([api().SetFocused(false).catch(() => {}), wait(UNFOCUS_TIMEOUT)]);
    runtime().Quit();
  };
  return {
    minimise: () => runtime().WindowMinimise(),
    toggleMaximise: () => (windowsApp ? runtime().WindowToggleMaximise() : void api().ToggleMaximise()),
    close: async () => {
      if (windowsApp) return closeApp();
      await api().Hide();
    },
    quitAll: async () => {
      if (!windowsApp) return api().Quit();
      if (ready()) {
        try {
          await api().Quit();
        } catch {}
      }
      runtime().Quit();
    },
    isMaximised: async () => (mode === 'browser' ? false : ((await runtime().WindowIsMaximised?.()) ?? false)),
  };
}
