import { expect, it } from 'vitest';
import { windowControls } from './windowControls';

function fakes(maximised?: boolean) {
  const calls: string[] = [];
  const api = {
    Hide: async () => { calls.push('api.Hide'); },
    ToggleMaximise: async () => { calls.push('api.ToggleMaximise'); },
    Quit: async () => { calls.push('api.Quit'); },
  };
  const runtime = {
    WindowMinimise: () => { calls.push('runtime.WindowMinimise'); },
    WindowToggleMaximise: () => { calls.push('runtime.WindowToggleMaximise'); },
    WindowIsMaximised: maximised === undefined ? undefined : async () => maximised,
    Quit: () => { calls.push('runtime.Quit'); },
  };
  return { calls, api: () => api, runtime: () => runtime };
}

it('the linux window hides on close and maximises through the bridge', async () => {
  const f = fakes(true);
  const c = windowControls('linux-window', f.api, f.runtime);
  c.minimise();
  c.toggleMaximise();
  c.close();
  await c.quitAll();
  expect(f.calls).toEqual(['runtime.WindowMinimise', 'api.ToggleMaximise', 'api.Hide', 'api.Quit']);
  await expect(c.isMaximised()).resolves.toBe(true);
});

it('the windows app closes only its own window and maximises through the runtime', async () => {
  const f = fakes(false);
  const c = windowControls('windows-app', f.api, f.runtime);
  c.minimise();
  c.toggleMaximise();
  c.close();
  expect(f.calls).toEqual(['runtime.WindowMinimise', 'runtime.WindowToggleMaximise', 'runtime.Quit']);
  await expect(c.isMaximised()).resolves.toBe(false);
});

it('quitting everything from the windows app stops the server and then closes the window', async () => {
  const f = fakes();
  await windowControls('windows-app', f.api, f.runtime).quitAll();
  expect(f.calls).toEqual(['api.Quit', 'runtime.Quit']);
});

it('the windows app still closes when the server quit fails', async () => {
  const f = fakes();
  const api = () => ({ ...f.api(), Quit: async () => { throw new Error('conexão com a skye perdida'); } });
  await windowControls('windows-app', api, f.runtime).quitAll();
  expect(f.calls).toEqual(['runtime.Quit']);
});

it('reports not maximised when the runtime cannot tell', async () => {
  const f = fakes();
  await expect(windowControls('windows-app', f.api, f.runtime).isMaximised()).resolves.toBe(false);
  await expect(windowControls('browser', f.api, f.runtime).isMaximised()).resolves.toBe(false);
});

it('the browser only quits', async () => {
  const f = fakes(true);
  await windowControls('browser', f.api, f.runtime).quitAll();
  expect(f.calls).toEqual(['api.Quit']);
});
