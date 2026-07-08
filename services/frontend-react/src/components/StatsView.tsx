import { useEffect, useState } from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { api } from "../api/client";
import type { DashboardStats, Status } from "../types";
import { STATUS_LABELS, STATUSES } from "../types";

const STATUS_COLORS: Record<Status, string> = {
  wishlist: "#94a3b8",
  applied: "#6366f1",
  oa: "#22d3ee",
  interview: "#fbbf24",
  offer: "#34d399",
  rejected: "#f87171",
};

export function StatsView() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .dashboardStats()
      .then(setStats)
      .catch((e) => setError(String(e)));
  }, []);

  if (error) return <p className="empty">Failed to load stats: {error}</p>;
  if (!stats) return <p className="empty">Loading…</p>;

  const statusData = STATUSES.map((s) => ({
    status: STATUS_LABELS[s],
    key: s,
    count: stats.counts_by_status[s],
  }));

  return (
    <>
      <div className="stat-tiles">
        <div className="tile">
          <div className="value">{stats.total_applications}</div>
          <div className="label">Total applications</div>
        </div>
        <div className="tile">
          <div className="value">{stats.submitted}</div>
          <div className="label">Submitted</div>
        </div>
        <div className="tile">
          <div className="value">{Math.round(stats.response_rate * 100)}%</div>
          <div className="label">Response rate</div>
        </div>
        <div className="tile">
          <div className="value">
            {stats.avg_match_score != null
              ? `${Math.round(stats.avg_match_score * 100)}%`
              : "—"}
          </div>
          <div className="label">Avg match score</div>
        </div>
      </div>

      <div className="stats">
        <div className="chart-card">
          <h3>Applications by status</h3>
          <ResponsiveContainer width="100%" height={260}>
            <BarChart data={statusData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
              <XAxis dataKey="status" stroke="#94a3b8" fontSize={12} />
              <YAxis allowDecimals={false} stroke="#94a3b8" fontSize={12} />
              <Tooltip
                contentStyle={{
                  background: "#1e293b",
                  border: "1px solid #334155",
                  borderRadius: 8,
                }}
              />
              <Bar dataKey="count" radius={[4, 4, 0, 0]}>
                {statusData.map((d) => (
                  <Cell key={d.key} fill={STATUS_COLORS[d.key]} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="chart-card">
          <h3>Applications over time</h3>
          {stats.applications_over_time.length === 0 ? (
            <p className="empty">No data yet</p>
          ) : (
            <ResponsiveContainer width="100%" height={260}>
              <LineChart data={stats.applications_over_time}>
                <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
                <XAxis dataKey="date" stroke="#94a3b8" fontSize={12} />
                <YAxis allowDecimals={false} stroke="#94a3b8" fontSize={12} />
                <Tooltip
                  contentStyle={{
                    background: "#1e293b",
                    border: "1px solid #334155",
                    borderRadius: 8,
                  }}
                />
                <Line
                  type="monotone"
                  dataKey="count"
                  stroke="#22d3ee"
                  strokeWidth={2}
                  dot={{ r: 3 }}
                />
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>
    </>
  );
}
