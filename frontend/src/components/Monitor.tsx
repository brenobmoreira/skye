import { useEffect, useState, type MouseEvent } from 'react';
import { api } from '../bridge';
import { itemLabel } from '../lib/label';
import { memoryLevel, nearest, size, sparkline, strayLabel } from '../lib/monitor';
import { stateLabel } from '../lib/states';
import type { MemoryPoint, MonitorReport, Terminal } from '../lib/types';
import { DogIcon } from './DogIcon';

const REFRESH_MS = 3000;
const CHART_W = 600;
const CHART_H = 64;

const clock = (iso: string) => new Date(iso).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });

function MemoryChart({ points, totalMb }: { points: MemoryPoint[]; totalMb: number }) {
  const [hover, setHover] = useState<number | null>(null);
  const d = sparkline(points, totalMb, CHART_W, CHART_H);
  if (!d) return null;
  const move = (e: MouseEvent<HTMLDivElement>) => {
    const box = e.currentTarget.getBoundingClientRect();
    setHover(nearest(points.length, (e.clientX - box.left) / box.width));
  };
  const at = hover === null ? null : points[hover];
  const x = hover === null ? 0 : (hover / (points.length - 1)) * 100;
  return (
    <figure className="mon-chart">
      <figcaption>Memória disponível no WSL · últimas 3 h</figcaption>
      <div className="plot" onMouseMove={move} onMouseLeave={() => setHover(null)}>
        <svg viewBox={`0 0 ${CHART_W} ${CHART_H}`} preserveAspectRatio="none" role="img" aria-label="memória disponível nas últimas 3 horas">
          <line x1="0" y1={CHART_H} x2={CHART_W} y2={CHART_H} className="axis" />
          <path d={d} className="series" />
        </svg>
        {at && (
          <>
            <span className="crosshair" style={{ left: `${x}%` }} />
            <span className={x > 70 ? 'tip left' : 'tip'} style={{ left: `${x}%` }}>
              {clock(at.t)} · {size(at.availMb)}
            </span>
          </>
        )}
      </div>
      <div className="ends">
        <span>{clock(points[0].t)}</span>
        <span>de 0 a {size(totalMb)}</span>
        <span>{clock(points[points.length - 1].t)}</span>
      </div>
    </figure>
  );
}

export function Monitor({ terminals, onOpen }: { terminals: Terminal[]; onOpen: (id: string) => void }) {
  const [report, setReport] = useState<MonitorReport | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    let alive = true;
    const load = () =>
      api()
        .Monitor()
        .then((r) => { if (alive) { setReport(r); setError(''); } })
        .catch((e) => { if (alive) setError(String(e?.message ?? e)); });
    load();
    const tick = setInterval(load, REFRESH_MS);
    return () => { alive = false; clearInterval(tick); };
  }, []);

  if (!report) return <div className="monitor"><p className="hint">{error || 'medindo…'}</p></div>;

  const m = report.machine;
  const level = memoryLevel(m);
  const byId = new Map(terminals.map((t) => [t.id, t]));
  const loads = [...(report.terminals ?? [])].sort((a, b) => b.rssMb - a.rssMb);
  const biggest = Math.max(1, ...loads.map((l) => l.rssMb));
  const total = loads.reduce((sum, l) => sum + l.rssMb, report.selfMb);
  const strays = report.strays ?? [];
  const servers = report.servers ?? [];
  const live = servers.filter((s) => s.alive);
  const dead = servers.filter((s) => !s.alive);
  const usedPct = m.memTotalMb > 0 ? Math.round(((m.memTotalMb - m.memAvailMb) / m.memTotalMb) * 100) : 0;
  const host = report.host;
  const hostLevel = host ? memoryLevel({ memTotalMb: host.totalMb, memAvailMb: host.availMb }) : 'ok';
  const hostUsedPct = host && host.totalMb > 0 ? Math.round(((host.totalMb - host.availMb) / host.totalMb) * 100) : 0;

  return (
    <div className="monitor">
      {error && <p className="hint warn">falhou a última leitura: {error}</p>}
      <section className="mon-tiles">
        {host ? (
          <div className="tile" title="o que o Gerenciador de Tarefas do Windows mostra como disponível; inclui o que a VM do WSL segura">
            <span className="label">Windows · disponível</span>
            <span className="value">{size(host.availMb)}</span>
            <span className="meter"><span className={hostLevel === 'ok' ? '' : 'warn'} style={{ width: `${hostUsedPct}%` }} /></span>
            <span className="note">
              {hostLevel !== 'ok' && <strong className="warn">⚠ memória {hostLevel} · </strong>}
              {hostUsedPct}% usada de {size(host.totalMb)}
            </span>
          </div>
        ) : null}
        <div className="tile" title="dentro da VM do WSL, até o teto do .wslconfig">
          <span className="label">{host ? 'WSL · disponível' : 'RAM disponível'}</span>
          <span className="value">{size(m.memAvailMb)}</span>
          <span className="meter"><span className={level === 'ok' ? '' : 'warn'} style={{ width: `${usedPct}%` }} /></span>
          <span className="note">
            {level !== 'ok' && <strong className="warn">⚠ memória {level} · </strong>}
            {usedPct}% usada de {size(m.memTotalMb)}{host ? ' (teto do WSL)' : ''}
          </span>
        </div>
        <div className="tile">
          <span className="label">Swap usada</span>
          <span className="value">{size(m.swapUsedMb)}</span>
          <span className="note">de {size(m.swapTotalMb)}</span>
        </div>
        <div className="tile">
          <span className="label">Pressão de memória</span>
          <span className="value">{m.psiSome60.toFixed(1).replace('.', ',')}%</span>
          <span className="note">tempo esperando memória (média de 60 s)</span>
        </div>
        <div className="tile">
          <span className="label">Carga</span>
          <span className="value">{m.load1.toFixed(2).replace('.', ',')}</span>
          <span className="note">média de 1 min</span>
        </div>
      </section>

      {report.history && report.history.length > 1 && <MemoryChart points={report.history} totalMb={m.memTotalMb} />}

      <section>
        <h3>Terminais da skye ({loads.length})</h3>
        <div className="mon-scroll">
        <table className="mon-table">
          <thead>
            <tr><th>terminal</th><th>estado</th><th className="num">processos</th><th className="num">memória</th><th className="bar-col" /></tr>
          </thead>
          <tbody>
            {loads.map((l) => {
              const t = byId.get(l.id);
              return (
                <tr key={l.id} className="link" onClick={() => onOpen(l.id)} title={`abrir · PID ${l.pid}`}>
                  <td>
                    <span className="name">
                      {t && <DogIcon color={`var(--${t.state})`} title={stateLabel[t.state]} />}
                      <span>{t ? itemLabel(t).name : l.id}</span>
                    </span>
                  </td>
                  <td>{t ? stateLabel[t.state] : ''}</td>
                  <td className="num">{l.procs}</td>
                  <td className="num">{size(l.rssMb)}</td>
                  <td className="bar-col"><span className="meter"><span style={{ width: `${(l.rssMb / biggest) * 100}%` }} /></span></td>
                </tr>
              );
            })}
            <tr className="muted">
              <td>a própria skye</td><td /><td /><td className="num">{size(report.selfMb)}</td><td />
            </tr>
          </tbody>
          <tfoot>
            <tr><td>total</td><td /><td /><td className="num">{size(total)}</td><td /></tr>
          </tfoot>
        </table>
        </div>
      </section>

      <section>
        <h3>Fora da lista ({strays.length})</h3>
        {strays.length === 0 ? (
          <p className="hint">Nada fora da lista: todo processo com a marca da skye está num terminal aberto.</p>
        ) : (
          <>
            <div className="mon-scroll">
            <table className="mon-table">
              <thead>
                <tr><th>o quê</th><th className="num">PID</th><th>comando</th><th>pasta</th><th className="num">processos</th><th className="num">memória</th></tr>
              </thead>
              <tbody>
                {strays.map((s) => (
                  <tr key={`${s.kind}:${s.pid}`}>
                    <td>{strayLabel(s.kind)}</td>
                    <td className="num">{s.pid}</td>
                    <td className="cmd" title={s.args}>{s.args || s.name}</td>
                    <td className="cmd" title={s.cwd}>{s.cwd}</td>
                    <td className="num">{s.procs}</td>
                    <td className="num">{size(s.rssMb)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            </div>
            <p className="hint">A skye só mostra; para encerrar, <code>kill &lt;PID&gt;</code> num terminal.</p>
          </>
        )}
      </section>

      {servers.length > 0 && (
        <section>
          <h3>Outros servidores tmux ({servers.length})</h3>
          <ul className="mon-servers">
            {live.map((s) => (
              <li key={s.name}><code>{s.name}</code> · vivo</li>
            ))}
            {dead.length > 0 && (
              <li>
                <details>
                  <summary>{dead.length} {dead.length === 1 ? 'morto' : 'mortos'}: só o arquivo do socket ficou, pode apagar</summary>
                  {dead.map((s) => <div key={s.name}><code>{s.name}</code></div>)}
                </details>
              </li>
            )}
          </ul>
        </section>
      )}
    </div>
  );
}
