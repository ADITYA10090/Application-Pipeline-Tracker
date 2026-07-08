import express from "express";
import "express-async-errors"; // forwards async route errors to the error handler
import cors from "cors";
import { config } from "./config";
import { pool } from "./db";
import { applicationsRouter } from "./routes/applications";
import { companiesRouter } from "./routes/companies";
import { jobsRouter } from "./routes/jobs";
import { matchRouter } from "./routes/match";
import { dashboardRouter } from "./routes/dashboard";

const app = express();
app.use(cors());
app.use(express.json({ limit: "2mb" }));

app.get("/health", (_req, res) => res.json({ status: "ok" }));

app.use("/api/applications", applicationsRouter);
app.use("/api/companies", companiesRouter);
app.use("/api/jobs", jobsRouter);
app.use("/api/match-score", matchRouter);
app.use("/api/dashboard", dashboardRouter);

// Central error handler so a thrown query error becomes a clean 500.
app.use(
  (
    err: unknown,
    _req: express.Request,
    res: express.Response,
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    _next: express.NextFunction
  ) => {
    console.error("unhandled error", err);
    res.status(500).json({ error: "internal server error" });
  }
);

const server = app.listen(config.port, () => {
  console.log(`api-gateway listening on :${config.port}`);
});

async function shutdown() {
  console.log("shutting down");
  server.close();
  await pool.end();
  process.exit(0);
}
process.on("SIGTERM", shutdown);
process.on("SIGINT", shutdown);
