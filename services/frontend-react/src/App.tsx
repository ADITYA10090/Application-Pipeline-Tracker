import { useCallback, useEffect, useState } from "react";
import { api } from "./api/client";
import type { Application, Status } from "./types";
import { KanbanBoard } from "./components/KanbanBoard";
import { ApplicationModal } from "./components/ApplicationModal";
import { AddJobModal } from "./components/AddJobModal";
import { StatsView } from "./components/StatsView";

type Tab = "board" | "stats";

export function App() {
  const [tab, setTab] = useState<Tab>("board");
  const [apps, setApps] = useState<Application[]>([]);
  const [openId, setOpenId] = useState<number | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const [toast, setToast] = useState<string | null>(null);
  const [scraping, setScraping] = useState(false);

  const flash = useCallback((msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(null), 2500);
  }, []);

  const load = useCallback(() => {
    api
      .listApplications()
      .then(setApps)
      .catch((e) => flash(`Load failed: ${e}`));
  }, [flash]);

  useEffect(() => {
    load();
  }, [load]);

  // Optimistic drag-to-update: move the card immediately, reconcile on error.
  async function move(id: number, status: Status) {
    const prev = apps;
    setApps((cur) =>
      cur.map((a) => (a.id === id ? { ...a, current_status: status } : a)),
    );
    try {
      await api.updateStatus(id, status);
    } catch (e) {
      setApps(prev);
      flash(`Update failed: ${e}`);
    }
  }

  async function triggerScrape() {
    setScraping(true);
    try {
      const r = await api.triggerScrape();
      flash(`Scrape ${r.status}`);
    } catch (e) {
      flash(`Scrape failed: ${e}`);
    } finally {
      setScraping(false);
    }
  }

  return (
    <>
      <header className="app-header">
        <h1>
          Track<span className="brand">Folio</span>
        </h1>
        <div className="tabs">
          <button
            className={`tab ${tab === "board" ? "active" : ""}`}
            onClick={() => setTab("board")}
          >
            Board
          </button>
          <button
            className={`tab ${tab === "stats" ? "active" : ""}`}
            onClick={() => setTab("stats")}
          >
            Stats
          </button>
        </div>
        <div className="spacer" />
        <button className="btn secondary" onClick={() => setShowAdd(true)}>
          + Add job
        </button>
        <button className="btn" disabled={scraping} onClick={triggerScrape}>
          {scraping ? "Triggering…" : "Trigger scrape"}
        </button>
      </header>

      {tab === "board" ? (
        apps.length === 0 ? (
          <p className="empty">
            No applications yet. Click “Add job” to pull one from your scraped
            postings, or “Trigger scrape” to fetch more.
          </p>
        ) : (
          <KanbanBoard
            applications={apps}
            onMove={move}
            onOpen={(a) => setOpenId(a.id)}
          />
        )
      ) : (
        <StatsView />
      )}

      {openId != null && (
        <ApplicationModal
          applicationId={openId}
          onClose={() => setOpenId(null)}
        />
      )}
      {showAdd && (
        <AddJobModal
          onClose={() => setShowAdd(false)}
          onAdded={() => {
            setShowAdd(false);
            load();
            flash("Added to pipeline");
          }}
        />
      )}
      {toast && <div className="toast">{toast}</div>}
    </>
  );
}
