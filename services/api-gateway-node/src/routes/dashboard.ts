import { Router } from "express";
import { query } from "../db";
import { STATUSES } from "../config";

export const dashboardRouter = Router();

// GET /api/dashboard/stats — counts by status, response rate, avg match score,
// and applications-created-over-time for the charts.
dashboardRouter.get("/stats", async (_req, res) => {
  const byStatusRows = await query<{ current_status: string; count: number }>(
    `SELECT current_status, COUNT(*)::int AS count
       FROM applications GROUP BY current_status`
  );
  const byStatus: Record<string, number> = {};
  for (const s of STATUSES) byStatus[s] = 0;
  for (const r of byStatusRows) byStatus[r.current_status] = r.count;

  const total = Object.values(byStatus).reduce((a, b) => a + b, 0);

  // "Responded" = moved beyond applied (heard back with OA/interview/offer/reject).
  const responded =
    byStatus["oa"] +
    byStatus["interview"] +
    byStatus["offer"] +
    byStatus["rejected"];
  const applied = total - byStatus["wishlist"];
  const responseRate = applied > 0 ? Number((responded / applied).toFixed(4)) : 0;

  const avgRows = await query<{ avg: number | null }>(
    `SELECT AVG(match_score)::float AS avg FROM applications WHERE match_score IS NOT NULL`
  );
  const avgMatchScore = avgRows[0]?.avg
    ? Number(avgRows[0].avg.toFixed(4))
    : 0;

  const overTime = await query(
    `SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD') AS day,
            COUNT(*)::int AS count
       FROM applications
      GROUP BY day ORDER BY day ASC`
  );

  res.json({
    total_applications: total,
    by_status: byStatus,
    response_rate: responseRate,
    avg_match_score: avgMatchScore,
    applications_over_time: overTime,
  });
});
