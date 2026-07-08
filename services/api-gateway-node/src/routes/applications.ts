import { Router } from "express";
import { query } from "../db";
import {
  createApplicationSchema,
  listApplicationsQuery,
  updateStatusSchema,
} from "../validation";

export const applicationsRouter = Router();

// GET /api/applications?status=&company=
applicationsRouter.get("/", async (req, res) => {
  const parsed = listApplicationsQuery.safeParse(req.query);
  if (!parsed.success) {
    return res.status(400).json({ error: parsed.error.flatten() });
  }
  const { status, company } = parsed.data;

  const clauses: string[] = [];
  const params: unknown[] = [];
  if (status) {
    params.push(status);
    clauses.push(`a.current_status = $${params.length}`);
  }
  if (company) {
    params.push(company);
    clauses.push(`c.name = $${params.length}`);
  }
  const where = clauses.length ? `WHERE ${clauses.join(" AND ")}` : "";

  const rows = await query(
    `SELECT a.id, a.job_posting_id, a.resume_version, a.applied_date,
            a.current_status, a.match_score, a.created_at, a.updated_at,
            jp.title, jp.url, jp.location, jp.mongo_jd_ref,
            c.name AS company
       FROM applications a
       JOIN job_postings jp ON jp.id = a.job_posting_id
       JOIN companies c ON c.id = jp.company_id
       ${where}
       ORDER BY a.updated_at DESC`,
    params
  );
  res.json(rows);
});

// POST /api/applications
applicationsRouter.post("/", async (req, res) => {
  const parsed = createApplicationSchema.safeParse(req.body);
  if (!parsed.success) {
    return res.status(400).json({ error: parsed.error.flatten() });
  }
  const d = parsed.data;

  const jp = await query(`SELECT id FROM job_postings WHERE id = $1`, [
    d.job_posting_id,
  ]);
  if (jp.length === 0) {
    return res.status(404).json({ error: "job_posting_id not found" });
  }

  const rows = await query(
    `INSERT INTO applications
       (job_posting_id, resume_version, applied_date, current_status, match_score)
     VALUES ($1,$2,$3,$4,$5)
     RETURNING *`,
    [
      d.job_posting_id,
      d.resume_version ?? null,
      d.applied_date ?? null,
      d.current_status,
      d.match_score ?? null,
    ]
  );
  const app = rows[0];
  await query(
    `INSERT INTO status_events (application_id, status, source)
     VALUES ($1,$2,'manual')`,
    [app.id, app.current_status]
  );
  res.status(201).json(app);
});

// PATCH /api/applications/:id/status
applicationsRouter.patch("/:id/status", async (req, res) => {
  const id = Number(req.params.id);
  if (!Number.isInteger(id)) {
    return res.status(400).json({ error: "invalid id" });
  }
  const parsed = updateStatusSchema.safeParse(req.body);
  if (!parsed.success) {
    return res.status(400).json({ error: parsed.error.flatten() });
  }
  const { status, source } = parsed.data;

  const rows = await query(
    `UPDATE applications
        SET current_status = $1, updated_at = now()
      WHERE id = $2
      RETURNING *`,
    [status, id]
  );
  if (rows.length === 0) {
    return res.status(404).json({ error: "application not found" });
  }
  await query(
    `INSERT INTO status_events (application_id, status, source)
     VALUES ($1,$2,$3)`,
    [id, status, source]
  );
  res.json(rows[0]);
});

// GET /api/applications/:id  (detail incl. status history)
applicationsRouter.get("/:id", async (req, res) => {
  const id = Number(req.params.id);
  if (!Number.isInteger(id)) {
    return res.status(400).json({ error: "invalid id" });
  }
  const rows = await query(
    `SELECT a.*, jp.title, jp.url, jp.location, jp.mongo_jd_ref, c.name AS company
       FROM applications a
       JOIN job_postings jp ON jp.id = a.job_posting_id
       JOIN companies c ON c.id = jp.company_id
      WHERE a.id = $1`,
    [id]
  );
  if (rows.length === 0) {
    return res.status(404).json({ error: "application not found" });
  }
  const events = await query(
    `SELECT status, source, occurred_at FROM status_events
      WHERE application_id = $1 ORDER BY occurred_at ASC`,
    [id]
  );
  res.json({ ...rows[0], status_events: events });
});
