import { useEffect, useState } from "react";
import { api } from "../api";
import type { ApplicationDetail } from "../types";
import { STATUS_LABELS } from "../types";

interface Props {
  id: number;
  onClose: () => void;
}

export function DetailModal({ id, onClose }: Props) {
  const [detail, setDetail] = useState<ApplicationDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.getApplication(id).then(setDetail).catch((e) => setError(String(e)));
  }, [id]);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <button className="modal-close" onClick={onClose}>
          ×
        </button>
        {error && <p className="error">{error}</p>}
        {!detail && !error && <p>Loading…</p>}
        {detail && (
          <>
            <h2>{detail.title}</h2>
            <p className="muted">
              {detail.company}
              {detail.location ? ` · ${detail.location}` : ""}
            </p>
            <p>
              <a href={detail.url} target="_blank" rel="noreferrer">
                View posting ↗
              </a>
            </p>
            <div className="detail-grid">
              <div>
                <strong>Status</strong>
                <div>{STATUS_LABELS[detail.current_status]}</div>
              </div>
              <div>
                <strong>Match score</strong>
                <div>
                  {detail.match_score != null
                    ? `${(detail.match_score * 100).toFixed(0)}%`
                    : "—"}
                </div>
              </div>
              <div>
                <strong>Resume</strong>
                <div>{detail.resume_version ?? "—"}</div>
              </div>
            </div>

            <h3>Status history</h3>
            <ul className="timeline">
              {detail.status_events.map((ev, i) => (
                <li key={i}>
                  <span className="dot" />
                  {STATUS_LABELS[ev.status]}{" "}
                  <span className="muted">
                    ({ev.source}) · {new Date(ev.occurred_at).toLocaleString()}
                  </span>
                </li>
              ))}
            </ul>
          </>
        )}
      </div>
    </div>
  );
}
