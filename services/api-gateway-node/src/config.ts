export const config = {
  port: parseInt(process.env.PORT ?? "8080", 10),
  databaseUrl:
    process.env.DATABASE_URL ??
    "postgres://postgres:postgres@localhost:5432/trackfolio",
  scraperUrl: process.env.SCRAPER_URL ?? "http://localhost:8081",
  matchServiceUrl: process.env.MATCH_SERVICE_URL ?? "http://localhost:8082",
};
