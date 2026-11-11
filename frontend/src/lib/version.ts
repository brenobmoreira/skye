// shouldReload: the server came back with another interface build than the one this page loaded
// with, so the page reloads to pick it up. An empty version is unknown and never reloads.
export function shouldReload(seen: string, now: string): boolean {
  return seen !== '' && now !== '' && seen !== now;
}
