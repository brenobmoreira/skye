// Breno's curled-up sleeping dog: a ball with two ears and the curve of the body.
const EARS = [
  'M178.2 176.3 C 142.2 100.5 147.0 50.7 147.0 50.7 C 147.0 50.7 191.0 74.5 233.8 146.8 Z',
  'M140.2 229.6 C 76.5 181.7 59.4 140.0 59.4 140.0 C 59.4 140.0 104.5 140.1 172.7 181.3 Z',
];
const SHADOW = 'M0 -170 A 170 170 0 0 1 0 170 A 85.0 85.0 0 0 1 0 0 A 85.0 85.0 0 0 0 0 -170 Z';
const CURVE = 'M0 170 A 85.0 85.0 0 0 1 0 0 A 85.0 85.0 0 0 0 0 -170';

const ears = (color: string) =>
  EARS.map((d) => <path key={d} d={d} fill={color} stroke={color} strokeWidth="14" strokeLinejoin="round" />);

// The state marker in the list: the one-color dog, tinted with the state's color. The curve is
// drawn thicker than in the logo so it still shows at 18px.
export function DogIcon({ color, title }: { color: string; title?: string }) {
  return (
    <svg className="dog" viewBox="40 36 424 424" style={{ color }} role="img" aria-label={title}>
      {ears('currentColor')}
      <circle cx="270" cy="282" r="170" fill="currentColor" />
      <g transform="translate(270 282) rotate(135)">
        <path d={CURVE} stroke="var(--bg)" strokeWidth="44" strokeLinecap="round" fill="none" />
      </g>
    </svg>
  );
}

// The full-color logo on its tile, as in public/icon-*.png.
export function Logo() {
  return (
    <svg className="logo" viewBox="0 0 512 512" role="img" aria-label="skye">
      <rect width="512" height="512" rx="116" fill="#efe3d3" />
      {ears('#7d4f33')}
      <circle cx="270" cy="282" r="170" fill="#171311" />
      <g transform="translate(270 282) rotate(135)">
        <path d={SHADOW} fill="#3b2a20" />
      </g>
    </svg>
  );
}
