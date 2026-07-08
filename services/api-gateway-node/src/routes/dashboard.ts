import { Router } from "express";
import { query } from "../db.js";
import { STATUSES } from "../schemas.js";
import { asyncHandler } from "../util.js";

export const dashboardRouter = Router();

/**
 * GET /api/dashboard/stats
 * Aggregates for the dashboard: counts per status, response rate, average
 * match score, and an applications-over-time series.
 *
 * response_rate definition: of applications that were actually submitted
 * (current_status != 'wishlist'), the fraction that received any response,
 * i.e. moved to oa / interview / offer / rejected.
 */
dashboardRouter.get(
  "/stats",
  asyncHandler(async (_req, res) => {
    const statusCounts = await query<{ current_status: string; n: string }>(
      `SELECT current_status, COUNT(*) AS n FROM applications GROUP BY current_status`,
    );
    const countsByStatus: Record<string, number> = {};
    for (const s of STATUSES) countsByStatus[s] = 0;
    for (const row of statusCounts.rows) {
      countsByStatus[row.current_status] = Number(row.n);
    }

    const submitted = STATUSES.filter((s) => s !== "wishlist").reduce(
      (acc, s) => acc + countsByStatus[s],
      0,
    );
    const responded =
      countsByStatus.oa +
      countsByStatus.interview +
      countsByStatus.offer +
      countsByStatus.rejected;
    const responseRate = submitted > 0 ? responded / submitted : 0;

    const avg = await query<{ avg: string | null }>(
      `SELECT AVG(match_score) AS avg FROM applications WHERE match_score IS NOT NULL`,
    );

    const overTime = await query<{ day: string; n: string }>(
      `SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD') AS day,
              COUNT(*) AS n
         FROM applications
         GROUP BY day ORDER BY day`,
    );

    const total = Object.values(countsByStatus).reduce((a, b) => a + b, 0);

    res.json({
      total_applications: total,
      counts_by_status: countsByStatus,
      submitted,
      response_rate: Number(responseRate.toFixed(4)),
      avg_match_score: avg.rows[0].avg ? Number(Number(avg.rows[0].avg).toFixed(4)) : null,
      applications_over_time: overTime.rows.map((r) => ({
        date: r.day,
        count: Number(r.n),
      })),
    });
  }),
);
