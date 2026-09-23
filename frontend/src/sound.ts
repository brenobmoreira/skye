let audio: HTMLAudioElement | null = null;
let toneCtx: AudioContext | null = null;

function beep() {
  toneCtx ??= new AudioContext();
  const ctx = toneCtx;
  if (ctx.state === 'suspended') ctx.resume();
  [0, 0.18].forEach((start) => {
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.type = 'square';
    osc.frequency.setValueAtTime(520, ctx.currentTime + start);
    osc.frequency.exponentialRampToValueAtTime(260, ctx.currentTime + start + 0.12);
    gain.gain.setValueAtTime(0.15, ctx.currentTime + start);
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + start + 0.14);
    osc.connect(gain).connect(ctx.destination);
    osc.start(ctx.currentTime + start);
    osc.stop(ctx.currentTime + start + 0.15);
  });
}

export async function bark() {
  try {
    audio ??= new Audio('bark.ogg');
    audio.currentTime = 0;
    await audio.play();
  } catch {
    beep();
  }
}
