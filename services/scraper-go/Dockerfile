# Multi-stage build: compile a static binary, ship it on scratch.
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO disabled -> fully static binary that runs on scratch.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/scraper .

FROM scratch
# CA certs so outbound HTTPS to the job-board APIs verifies.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/scraper /scraper
EXPOSE 8081
# No shell in scratch: liveness is via the /health HTTP endpoint (k8s httpGet).
ENTRYPOINT ["/scraper"]
