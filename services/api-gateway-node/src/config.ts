export const config = {
  port: parseInt(process.env.PORT ?? "8080", 10),
  databaseUrl:
    process.env.DATABASE_URL ??
    "postgres://trackfolio:trackfolio@postgres:5432/trackfolio",
  matchServiceUrl: process.env.MATCH_SERVICE_URL ?? "http://match-service:8000",
  scraperUrl: process.env.SCRAPER_URL ?? "http://scraper:8081",
};

// The canonical application pipeline. Order matters for the Kanban board.
export const STATUSES = [
  "wishlist",
  "applied",
  "oa",
  "interview",
  "offer",
  "rejected",
] as const;

export type Status = (typeof STATUSES)[number];
