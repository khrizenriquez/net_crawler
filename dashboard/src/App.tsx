import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "./api";
import { coverage, formatBytes, when } from "./format";
import type { Capture, Device, Finding, Metric, Radio, Schedule, Status } from "./types";

type Data = { status: Status; captures: Capture[]; metrics: Metric[]; devices: Device[]; findings: Finding[]; radios: Radio[]; schedules: Schedule[] };
type Page = "resumen" | "consumo" | "sesiones" | "dispositivos" | "hallazgos" | "horarios" | "configuracion" | "sistema";
const nav: Array<[Page, string, string]> = [
  ["resumen", "Resumen", "01"], ["consumo", "Consumo", "02"], ["sesiones", "Sesiones", "03"], ["dispositivos", "Dispositivos", "04"],
  ["hallazgos", "Hallazgos", "05"], ["horarios", "Horarios", "06"], ["configuracion", "Configuración", "07"], ["sistema", "Sistema", "08"],
];
export function App() {
  const [page, setPage] = useState<Page>("resumen");
  const [data, setData] = useState<Data>();
  const [error, setError] = useState("");
  const refresh = useCallback(async () => { try { setData(await api.load()); setError(""); } catch { setError("La API local no responde. Inicia el laboratorio desde la app de barra de menú."); } }, []);
  useEffect(() => { void refresh(); const events = api.events(); events.onmessage = () => void refresh(); events.onerror = () => setError("Reconectando con la instrumentación local…"); return () => events.close(); }, [refresh]);
  if (!data) return <Boot error={error} retry={refresh} />;
  return (
    <div className="shell">
      <aside className="rail">
        <div className="brand"><span className="brand-mark">D</span><div><strong>DUKU</strong><small>NET LAB / LOCAL</small></div></div>
        <nav>{nav.map(([id, label, number]) => <button key={id} className={page === id ? "nav active" : "nav"} onClick={() => setPage(id)}><span>{number}</span>{label}</button>)}</nav>
        <div className="rail-foot"><i className="pulse" /> DATOS LOCALES<br /><small>127.0.0.1 / SIN NUBE</small></div>
      </aside>
      <main>
        <header><div><p className="eyebrow">INSTRUMENTACIÓN INALÁMBRICA</p><h1>{nav.find(([id]) => id === page)?.[1]}</h1></div><div className="header-status"><span className="live">● EN LÍNEA</span><span>{new Date().toLocaleTimeString("es-GT", { hour: "2-digit", minute: "2-digit" })}</span></div></header>
        {error ? <div className="banner">{error}</div> : null}
        {page === "resumen" ? <Overview data={data} refresh={refresh} /> : null}
        {page === "consumo" ? <Consumption metrics={data.metrics} /> : null}
        {page === "sesiones" ? <Sessions captures={data.captures} /> : null}
        {page === "dispositivos" ? <Devices devices={data.devices} /> : null}
        {page === "hallazgos" ? <Findings findings={data.findings} /> : null}
        {page === "horarios" ? <Schedules schedules={data.schedules} /> : null}
        {page === "configuracion" ? <Configuration radios={data.radios} refresh={refresh} /> : null}
        {page === "sistema" ? <System status={data.status} refresh={refresh} /> : null}
      </main>
    </div>
  );
}
function Boot({ error, retry }: { error: string; retry: () => Promise<void> }) { return <div className="boot"><div className="brand-mark big">D</div><p className="eyebrow">DUKU NET LAB</p><h1>Buscando instrumentación local</h1><p>{error || "Sincronizando el estado del laboratorio…"}</p><button onClick={() => void retry()}>Reintentar enlace</button></div> }
function Overview({ data, refresh }: { data: Data; refresh: () => Promise<void> }) {
  const wifi = data.metrics.filter(m => m.source === "wifi_observed").reduce((sum, m) => sum + m.bytesUp + m.bytesDown, 0);
  const wan = data.metrics.filter(m => m.source === "wan_total").reduce((sum, m) => sum + m.bytesUp + m.bytesDown, 0);
  return <><section className="status-grid">
    <Card title="Motor de captura" value={data.status.activeSession ? "CAPTURANDO" : "EN ESPERA"} note="Modo monitor temporal" tone="mint" />
    <Card title="Canal actual" value={data.status.activeSession ? String(data.status.activeSession.channel) : "—"} note="Rotación sujeta al probe del host" />
    <Card title="Wi-Fi observado" value={formatBytes(wifi)} note="Muestra parcial autorizada" tone="amber" />
    <Card title="WAN total" value={wan ? formatBytes(wan) : "NO DISP."} note={`Adaptador Huawei / ${data.status.routerStatus}`} />
  </section><section className="overview-grid"><Panel title="Espectro de sesión" tag="COBERTURA EN TIEMPO REAL"><Spectrum /></Panel><Panel title="Hallazgos recientes" tag={`${data.findings.length} EVENTOS REDACTADOS`}><FindingList findings={data.findings.slice(0, 4)} /></Panel></section>
  <section className="overview-grid lower"><Panel title="Radios autorizadas" tag="BSSID CONFIRMADOS"><RadioList radios={data.radios} /></Panel><Panel title="Control manual" tag="HELPER LIMITADO"><div className="control"><p>La captura temporal puede interrumpir la conectividad Wi-Fi normal de esta Mac.</p><div><button onClick={() => void api.start(data.radios[0]?.channel ?? 2).then(refresh)}>Iniciar captura</button><button className="ghost" onClick={() => void api.stop().then(refresh)}>Detener</button></div></div></Panel></section></>;
}
function Card({ title, value, note, tone = "" }: { title: string; value: string; note: string; tone?: string }) { return <article className={`card ${tone}`}><p>{title}</p><strong>{value}</strong><small>{note}</small></article> }
function Panel({ title, tag, children }: { title: string; tag: string; children: React.ReactNode }) { return <section className="panel"><div className="panel-head"><h2>{title}</h2><span>{tag}</span></div>{children}</section> }
function Spectrum() { return <div className="spectrum"><div className="scanline" />{Array.from({ length: 18 }, (_, i) => <span key={i} style={{ height: `${22 + ((i * 37) % 69)}%` }} />)}<footer><b>2.4 GHz / CH 02</b><em>muestra autorizada</em><b>5 GHz / CH 36</b></footer></div> }
function FindingList({ findings }: { findings: Finding[] }) { return <div className="event-list">{findings.map(f => <div className="event" key={f.id}><i className={f.severity} /><div><strong>{f.category.replaceAll("_", " ")}</strong><p>{f.redactedSample}</p></div><time>{when(f.createdAt)}</time></div>)}</div> }
function RadioList({ radios }: { radios: Radio[] }) { return <div className="radio-list">{radios.map(r => <div key={r.id}><span className="radio-wave">◉</span><div><strong>{r.ssid}</strong><small>{r.bssid}</small></div><b>{r.band}<br />CH {String(r.channel).padStart(2, "0")}</b></div>)}</div> }
function Consumption({ metrics }: { metrics: Metric[] }) { const max = Math.max(...metrics.map(m => m.bytesUp + m.bytesDown), 1); return <Panel title="Consumo por intervalo" tag="FUENTES SEPARADAS"><div className="bars">{metrics.map(m => <div key={m.id}><label>{when(m.timestamp)}</label><span><i style={{ width: `${((m.bytesUp + m.bytesDown) / max) * 100}%` }} /></span><b>{formatBytes(m.bytesUp + m.bytesDown)}</b><em>{coverage(m.source)}</em></div>)}</div></Panel> }
function Sessions({ captures }: { captures: Capture[] }) { return <Table heads={["Sesión", "Origen", "Estado", "Canal", "Cobertura", "Inicio", "Bytes"]} rows={captures.map(c => [c.id, c.origin, c.status, `${c.band} / ${c.channel}`, coverage(c.coverage), when(c.startedAt), formatBytes(c.bytesObserved)])} /> }
function Devices({ devices }: { devices: Device[] }) { return <Table heads={["Alias", "MAC local", "Bandas", "Última observación", "Bytes observados"]} rows={devices.map(d => [d.alias || "Sin alias", d.mac, d.bands.join(", "), when(d.lastSeenAt), formatBytes(d.bytesObserved)])} /> }
function Findings({ findings }: { findings: Finding[] }) { return <Panel title="Hallazgos redactados" tag="SIN SECRETOS PERSISTIDOS"><FindingList findings={findings} /></Panel> }
function Schedules({ schedules }: { schedules: Schedule[] }) { return <Table heads={["Ventana", "Días", "Inicio", "Duración", "Canales", "Rotación", "Estado"]} rows={schedules.map(s => [s.name, s.days.join(", "), `${Math.floor(s.startMinute / 60)}:${String(s.startMinute % 60).padStart(2, "0")}`, `${s.durationMinutes} min`, s.channels.join(", "), `${s.rotationMinutes} min`, s.enabled ? "Activa" : "Pausada"])} /> }
function Configuration({ radios, refresh }: { radios: Radio[]; refresh: () => Promise<void> }) { return <div className="overview-grid"><Panel title="Radios autorizadas" tag="ALLOWLIST BSSID"><RadioList radios={radios} /></Panel><Panel title="Modo demo" tag="FIXTURES SINTÉTICOS"><div className="control"><p>Los agentes trabajan únicamente con datos ficticios. Puedes restaurarlos sin tocar capturas reales.</p><div><button onClick={() => void api.seedDemo().then(refresh)}>Sembrar demo</button><button className="ghost" onClick={() => void api.resetDemo().then(refresh)}>Vaciar demo</button></div></div></Panel></div> }
function System({ status, refresh }: { status: Status; refresh: () => Promise<void> }) { return <section className="status-grid"><Card title="API local" value="ACTIVA" note="127.0.0.1:8080" tone="mint" /><Card title="PCAP filtrado" value={`${Math.round((status.pcapUsedBytes / status.pcapQuotaBytes) * 100)}%`} note={`${formatBytes(status.pcapUsedBytes)} / ${formatBytes(status.pcapQuotaBytes)}`} /><Card title="Colector Huawei" value={status.routerStatus.toUpperCase()} note="Consulta opcional / 15 min" tone="amber" /><article className="card"><p>Sincronización</p><button onClick={() => void refresh()}>Actualizar estado</button><small>{when(status.lastUpdatedAt)}</small></article></section> }
function Table({ heads, rows }: { heads: string[]; rows: string[][] }) { return <div className="table-wrap"><table><thead><tr>{heads.map(h => <th key={h}>{h}</th>)}</tr></thead><tbody>{rows.map((row, index) => <tr key={index}>{row.map((cell, i) => <td key={`${index}-${i}`}>{cell}</td>)}</tr>)}</tbody></table></div> }
