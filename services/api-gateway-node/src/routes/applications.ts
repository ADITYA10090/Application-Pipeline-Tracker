import { Router } from "express";
import { query, withTransaction } from "../db.js";
import {
  createApplicationSchema,
  listApplicationsQuerySchema,
  updateStatusSchema,
} from "../schemas.js";
import { asyncHandler, HttpError } from "../util.js";

export const applicationsRouter = Router();

/**
 * GET /api/applications?status=&company=
 * Lists applications joined to their posting + company, newest first.
 */
applicationsRouter.get(
  "/",
  asyncHandler(async (req, res) => {
    const q = listApplicationsQuerySchema.parse(req.query);
    const clauses: string[] = [];
    const params: unknown[] = [];
    if (q.status) {
      params.push(q.status);
      clauses.push(`a.current_status = $${params.length}`);
    }
    if (q.company) {
      params.push(q.company);
      clauses.push(`c.name = $${params.length}`);
    }
    const where = clauses.length ? `WHERE ${clauses.join(" AND ")}` : "";
    const { rows } = await query(
      `SELECT a.id, a.job_posting_id, a.current_status, a.match_score,
              a.resume_version, a.applied_date, a.created_at, a.updated_at,
              j.title, j.url, j.location, j.mongo_jd_ref,
              c.name AS company
         FROM applications a
         JOIN job_postings j ON j.id = a.job_posting_id
         JOIN companies c ON c.id = j.company_id
         ${where}
         ORDER BY a.updated_at DESC`,
      params,
    );
    res.json(rows);
  }),
);

/**
 * POST /api/applications
 * Creates an application for a job posting and records the initial status event.
 */
applicationsRouter.post(
  "/",
  asyncHandler(async (req, res) => {
    const body = createApplicationSchema.parse(req.body);

    const posting = await query(`SELECT id FROM job_postings WHERE id = $1`, [
      body.job_posting_id,
    ]);
    if (posting.rowCount === 0) {
      throw new HttpError(404, "job_posting not found");
    }

    const created = await withTransaction(async (client) => {
      const dup = await client.query(
        `SELECT id FROM applications WHERE job_posting_id = $1`,
        [body.job_posting_id],
      );
      if (dup.rowCount && dup.rowCount > 0) {
        throw new HttpError(409, "application already exists for this posting");
      }
      const insert = await client.query(
        `INSERT INTO applications
           (job_posting_id, resume_version, applied_date, current_status, match_score)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING *`,
        [
          body.job_posting_id,
          body.resume_version ?? null,
          body.applied_date ?? null,
          body.current_status,
          body.match_score ?? null,
        ],
      );
      const app = insert.rows[0];
      await client.query(
        `INSERT INTO status_events (application_id, status, source)
         VALUES ($1, $2, 'manual')`,
        [app.id, body.current_status],
      );
      return app;
    });

    res.status(201).json(created);
  }),
);

/**
 * PATCH /api/applications/:id/status
 * Updates the current status and appends a status_events row atomically.
 */
applicationsRouter.patch(
  "/:id/status",
  asyncHandler(async (req, res) => {
    const id = Number(req.params.id);
    if (!Number.isInteger(id)) throw new HttpError(400, "invalid id");
    const body = updateStatusSchema.parse(req.body);

    const updated = await withTransaction(async (client) => {
      const upd = await client.query(
        `UPDATE applications
            SET current_status = $1, updated_at = now()
          WHERE id = $2
          RETURNING *`,
        [body.status, id],
      );
      if (upd.rowCount === 0) throw new HttpError(404, "application not found");
      await client.query(
        `INSERT INTO status_events (application_id, status, source)
         VALUES ($1, $2, $3)`,
        [id, body.status, body.source],
      );
      return upd.rows[0];
    });

    res.json(updated);
  }),
);

/** GET /api/applications/:id — detail incl. status history. */
applicationsRouter.get(
  "/:id",
  asyncHandler(async (req, res) => {
    const id = Number(req.params.id);
    if (!Number.isInteger(id)) throw new HttpError(400, "invalid id");
    const { rows } = await query(
      `SELECT a.*, j.title, j.url, j.location, j.mongo_jd_ref, c.name AS company
         FROM applications a
         JOIN job_postings j ON j.id = a.job_posting_id
         JOIN companies c ON c.id = j.company_id
        WHERE a.id = $1`,
      [id],
    );
    if (rows.length === 0) throw new HttpError(404, "application not found");
    const events = await query(
      `SELECT status, source, occurred_at
         FROM status_events WHERE application_id = $1 ORDER BY occurred_at`,
      [id],
    );
    res.json({ ...rows[0], status_events: events.rows });
  }),
);
