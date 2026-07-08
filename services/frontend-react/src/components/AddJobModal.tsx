import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { Job } from "../types";

interface Props {
  onClose: () => void;
  onAdded: () => void;
}

/** Lists scraped postings without an application yet and lets you add one. */
export function AddJobModal({ onClose, onAdded }: Props) {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<number | null>(null);

  useEffect(() => {
    api
      .listJobs(true)
      .then(setJobs)
      .catch((e) => setError(String(e)));
  }, []);

  async function add(job: Job) {
    setBusyId(job.id);
    setError(null);
    try {
      await api.createApplication({
        job_posting_id: job.id,
        current_status: "wishlist",
      });
      onAdded();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusyId(null);
    }
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <button className="close" onClick={onClose}>
          ×
        </button>
        <h2>Add a job to the pipeline</h2>
        <p className="meta">Scraped postings not yet in your board.</p>
        {error && <p style={{ color: "var(--danger)" }}>{error}</p>}
        {jobs.length === 0 ? (
          <p className="empty">
            No un-applied postings. Trigger a scrape to fetch more.
          </p>
        ) : (
          jobs.map((j) => (
            <div
              key={j.id}
              style={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "center",
                borderBottom: "1px solid var(--border)",
                padding: "8px 0",
              }}
            >
              <div>
                <div style={{ fontWeight: 600 }}>{j.title}</div>
                <div className="meta">
                  {j.company}
                  {j.location ? ` · ${j.location}` : ""}
                </div>
              </div>
              <button
                className="btn secondary"
                disabled={busyId === j.id}
                onClick={() => add(j)}
              >
                {busyId === j.id ? "Adding…" : "Add"}
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
