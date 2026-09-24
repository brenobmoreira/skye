const HEAD = 'M6.4 9.4C6.4 6.4 8.9 4.3 12 4.3s5.6 2.1 5.6 5.1v4.6c0 3.4-2.5 6-5.6 6s-5.6-2.6-5.6-6z';
const EARS = 'M7.6 5C4.8 4.6 2.8 6.6 3.1 10.6c.2 3 1.3 5.1 2.8 5.3 1.1.1 1.6-1.6 1.6-3.8zM16.4 5c2.8-.4 4.8 1.6 4.5 5.6-.2 3-1.3 5.1-2.8 5.3-1.1.1-1.6-1.6-1.6-3.8z';

// The state marker in the list: one dog, tinted with the state's color. The app icons in
// public/ are the same drawing.
export function DogIcon({ color, title }: { color: string; title?: string }) {
  return (
    <svg className="dog" viewBox="0 0 24 24" style={{ color }} role="img" aria-label={title}>
      <path fill="currentColor" d={HEAD} />
      <path fill="currentColor" d={EARS} />
      <path fill="#000" opacity="0.28" d={EARS} />
      <circle cx="9.7" cy="10.9" r="1.1" fill="var(--bg)" />
      <circle cx="14.3" cy="10.9" r="1.1" fill="var(--bg)" />
      <ellipse cx="12" cy="15.7" rx="2.7" ry="2.1" fill="#fff" opacity="0.35" />
      <ellipse cx="12" cy="14.7" rx="1.35" ry="0.95" fill="var(--bg)" />
    </svg>
  );
}
