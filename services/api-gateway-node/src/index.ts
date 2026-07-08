import express, {
  type NextFunction,
  type Request,
  type Response,
} from "express";
import cors from "cors";
import { ZodError } from "zod";
import { config } from "./config.js";
import { pool } from "./db.js";
import { HttpError } from "./util.js";
import { applicationsRouter } from "./routes/applications.js";
import { jobsRouter } from "./routes/jobs.js";
import { companiesRouter } from "./routes/companies.js";
import { matchRouter } from "./routes/match.js";
import { dashboardRouter } from "./routes/dashboard.js";

export function createApp() {
  const app = express();
  app.use(cors());
  app.use(express.json({ limit: "1mb" }));

  app.get("/health", async (_req, res) => {
    try {
      await pool.query("SELECT 1");
      res.json({ status: "ok" });
    } catch {
      res.status(503).json({ status: "degraded", db: "unreachable" });
    }
  });

  app.use("/api/applications", applicationsRouter);
  app.use("/api/jobs", jobsRouter);
  app.use("/api/companies", companiesRouter);
  app.use("/api/match-score", matchRouter);
  app.use("/api/dashboard", dashboardRouter);

  // Centralized error handling: zod -> 400, HttpError -> its status, else 500.
  app.use((err: unknown, _req: Request, res: Response, _next: NextFunction) => {
    if (err instanceof ZodError) {
      return res.status(400).json({ error: "validation_error", details: err.errors });
    }
    if (err instanceof HttpError) {
      return res.status(err.status).json({ error: err.message });
    }
    console.error("unhandled error", err);
    return res.status(500).json({ error: "internal_error" });
  });

  return app;
}

// Start listening unless imported by tests (which set NODE_ENV=test).
if (process.env.NODE_ENV !== "test") {
  const app = createApp();
  app.listen(config.port, () => {
    console.log(
      JSON.stringify({
        msg: "api-gateway listening",
        port: config.port,
        scraper: config.scraperUrl,
        match: config.matchServiceUrl,
      }),
    );
  });
}
