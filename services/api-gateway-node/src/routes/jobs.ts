import { Router } from "express";
import { config } from "../config";
import { query } from "../db";

export const jobsRouter = Router();

// GET /api/jobs — scraped postings not yet turned into applications.
jobsRouter.get("/", async (_req, res) => {
  const rows = await query(
    `SELECT jp.id, jp.title, jp.url, jp.location, jp.posted_date,
            jp.scraped_at, jp.mongo_jd_ref, c.name AS company
       FROM job_postings jp
       JOIN companies c ON c.id = jp.company_id
      WHERE jp.id NOT IN (SELECT job_posting_id FROM applications)
      ORDER BY jp.scraped_at DESC
      LIMIT 200`
  );
  res.json(rows);
});

// POST /api/jobs/scrape-trigger — proxy to the Go scraper's /trigger.
jobsRouter.post("/scrape-trigger", async (_req, res) => {
  try {
    const r = await fetch(`${config.scraperUrl}/trigger`, { method: "POST" });
    const body = await r.json().catch(() => ({}));
    res.status(r.status).json(body);
  } catch (err) {
    res
      .status(502)
      .json({ error: "scraper unreachable", detail: String(err) });
  }
});
