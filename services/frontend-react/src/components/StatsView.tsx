import {
  Bar,
  BarChart,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { DashboardStats } from "../types";
import { STATUS_LABELS, STATUSES } from "../types";

export function StatsView({ stats }: { stats: DashboardStats }) {
  const byStatusData = STATUSES.map((s) => ({
    status: STATUS_LABELS[s],
    count: stats.by_status[s] ?? 0,
  }));

  return (
    <div className="stats">
      <div className="stat-tiles">
        <div className="tile">
          <div className="tile-value">{stats.total_applications}</div>
          <div className="tile-label">Applications</div>
        </div>
        <div className="tile">
          <div className="tile-value">
            {(stats.response_rate * 100).toFixed(0)}%
          </div>
          <div className="tile-label">Response rate</div>
        </div>
        <div className="tile">
          <div className="tile-value">
            {(stats.avg_match_score * 100).toFixed(0)}%
          </div>
          <div className="tile-label">Avg match score</div>
        </div>
      </div>

      <div className="chart-row">
        <div className="chart-card">
          <h3>Applications by status</h3>
          <ResponsiveContainer width="100%" height={240}>
            <BarChart data={byStatusData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="status" />
              <YAxis allowDecimals={false} />
              <Tooltip />
              <Bar dataKey="count" fill="#4f8cff" />
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="chart-card">
          <h3>Applications over time</h3>
          <ResponsiveContainer width="100%" height={240}>
            <LineChart data={stats.applications_over_time}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="day" />
              <YAxis allowDecimals={false} />
              <Tooltip />
              <Line
                type="monotone"
                dataKey="count"
                stroke="#22c55e"
                strokeWidth={2}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
}
