# --- build stage ---
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
# Static binary so the runtime image can be tiny.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/scraper .

# --- runtime stage ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
USER app
COPY --from=build /out/scraper /usr/local/bin/scraper
EXPOSE 8081
ENTRYPOINT ["scraper"]
