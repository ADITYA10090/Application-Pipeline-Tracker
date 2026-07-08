import { useCallback, useEffect, useState } from "react";
import { api } from "./api";
import type { Application, DashboardStats, Status } from "./types";
import { KanbanBoard } from "./components/KanbanBoard";
import { StatsView } from "./components/StatsView";
import { DetailModal } from "./components/DetailModal";

type Tab = "board" | "stats";

export default function App() {
  const [tab, setTab] = useState<Tab>("board");
  const [apps, setApps] = useState<Application[]>([]);
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [openId, setOpenId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [scraping, setScraping] = useState(false);

  const load = useCallback(() => {
    api.listApplications().then(setApps).catch((e) => setError(String(e)));
    api.getStats().then(setStats).catch((e) => setError(String(e)));
  }, []);

  useEffect(load, [load]);

  const handleMove = async (id: number, status: Status) => {
    // Optimistic update, then reconcile from the server.
    setApps((prev) =>
      prev.map((a) => (a.id === id ? { ...a, current_status: status } : a))
    );
    try {
      await api.updateStatus(id, status);
      load();
    } catch (e) {
      setError(String(e));
      load(); // roll back to server truth
    }
  };

  const handleScrape = async () => {
    setScraping(true);
    try {
      await api.triggerScrape();
    } catch (e) {
      setError(String(e));
    } finally {
      setScraping(false);
    }
  };

  return (
    <div className="app">
      <header className="topbar">
        <h1>TrackFolio</h1>
        <nav>
          <button
            className={tab === "board" ? "active" : ""}
            onClick={() => setTab("board")}
          >
            Board
          </button>
          <button
            className={tab === "stats" ? "active" : ""}
            onClick={() => setTab("stats")}
          >
            Stats
          </button>
        </nav>
        <div className="spacer" />
        <button className="primary" onClick={handleScrape} disabled={scraping}>
          {scraping ? "Triggering…" : "Trigger Scrape"}
        </button>
        <button onClick={load}>Refresh</button>
      </header>

      {error && (
        <div className="error-banner" onClick={() => setError(null)}>
          {error} (click to dismiss)
        </div>
      )}

      <main>
        {tab === "board" && (
          <KanbanBoard
            applications={apps}
            onMove={handleMove}
            onOpen={setOpenId}
          />
        )}
        {tab === "stats" &&
          (stats ? <StatsView stats={stats} /> : <p>Loading stats…</p>)}
      </main>

      {openId != null && (
        <DetailModal id={openId} onClose={() => setOpenId(null)} />
      )}
    </div>
  );
}
