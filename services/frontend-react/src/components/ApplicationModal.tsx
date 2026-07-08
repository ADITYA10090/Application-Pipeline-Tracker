import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { ApplicationDetail, MatchResult } from "../types";
import { STATUS_LABELS } from "../types";

interface Props {
  applicationId: number;
  onClose: () => void;
}

export function ApplicationModal({ applicationId, onClose }: Props) {
  const [detail, setDetail] = useState<ApplicationDetail | null>(null);
  const [resume, setResume] = useState("");
  const [jd, setJd] = useState("");
  const [match, setMatch] = useState<MatchResult | null>(null);
  const [scoring, setScoring] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getApplication(applicationId)
      .then(setDetail)
      .catch((e) => setError(String(e)));
  }, [applicationId]);

  async function runMatch() {
    setScoring(true);
    setError(null);
    try {
      setMatch(await api.matchScore(resume, jd));
    } catch (e) {
      setError(String(e));
    } finally {
      setScoring(false);
    }
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <button className="close" onClick={onClose}>
          ×
        </button>
        {!detail ? (
          <p className="empty">Loading…</p>
        ) : (
          <>
            <h2>{detail.title}</h2>
            <div className="meta">
              {detail.company}
              {detail.location ? ` · ${detail.location}` : ""} ·{" "}
              <a href={detail.url} target="_blank" rel="noreferrer">
                posting ↗
              </a>
              <br />
              Status: <strong>{STATUS_LABELS[detail.current_status]}</strong>
              {detail.match_score != null &&
                ` · stored match ${Math.round(detail.match_score * 100)}%`}
            </div>

            <h3>Status history</h3>
            <ul className="timeline">
              {detail.status_events.map((e, i) => (
                <li key={i}>
                  {new Date(e.occurred_at).toLocaleString()} →{" "}
                  {STATUS_LABELS[e.status as keyof typeof STATUS_LABELS] ??
                    e.status}{" "}
                  ({e.source})
                </li>
              ))}
            </ul>

            <h3>Match analysis</h3>
            <p className="meta">
              Paste your resume and the job description to score the fit and see
              the keyword gap (proxied to the Python match service).
            </p>
            <label>Resume text</label>
            <textarea
              rows={4}
              value={resume}
              onChange={(e) => setResume(e.target.value)}
            />
            <label>Job description text</label>
            <textarea
              rows={4}
              value={jd}
              onChange={(e) => setJd(e.target.value)}
            />
            <div style={{ marginTop: 10 }}>
              <button
                className="btn"
                disabled={scoring || !resume || !jd}
                onClick={runMatch}
              >
                {scoring ? "Scoring…" : "Score match"}
              </button>
            </div>

            {match && (
              <div style={{ marginTop: 14 }}>
                <div>
                  <strong>Score: {Math.round(match.score * 100)}%</strong>
                </div>
                <div style={{ marginTop: 8 }}>
                  <div className="meta">Matched keywords</div>
                  {match.matched_keywords.length === 0 && (
                    <span className="meta">none</span>
                  )}
                  {match.matched_keywords.map((k) => (
                    <span key={k} className="kw match">
                      {k}
                    </span>
                  ))}
                </div>
                <div style={{ marginTop: 8 }}>
                  <div className="meta">Missing keywords</div>
                  {match.missing_keywords.length === 0 && (
                    <span className="meta">none</span>
                  )}
                  {match.missing_keywords.map((k) => (
                    <span key={k} className="kw miss">
                      {k}
                    </span>
                  ))}
                </div>
              </div>
            )}
            {error && (
              <p style={{ color: "var(--danger)", fontSize: 13 }}>{error}</p>
            )}
          </>
        )}
      </div>
    </div>
  );
}
