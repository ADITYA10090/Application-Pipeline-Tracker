#!/usr/bin/env bash
# Fire concurrent scrape triggers at the Go scraper to exercise the worker pool
# and drive CPU up so the HPA scales the deployment.
#
# Usage:
#   TARGET=http://localhost:8081 CONCURRENCY=50 REQUESTS=2000 ./infra/loadtest.sh
#
# Against minikube, first expose the scraper:
#   kubectl -n trackfolio port-forward svc/scraper 8081:8081
# and in another terminal watch scaling:
#   kubectl -n trackfolio get hpa scraper -w
#   kubectl -n trackfolio get pods -l app=scraper -w

set -euo pipefail
TARGET="${TARGET:-http://localhost:8081}"
CONCURRENCY="${CONCURRENCY:-50}"
REQUESTS="${REQUESTS:-2000}"

echo "Load test: $REQUESTS requests, concurrency $CONCURRENCY -> $TARGET/trigger"
start=$(date +%s.%N)

seq "$REQUESTS" | xargs -P "$CONCURRENCY" -I{} \
  curl -s -o /dev/null -w "%{http_code}\n" -X POST "$TARGET/trigger" \
  | sort | uniq -c

end=$(date +%s.%N)
elapsed=$(echo "$end - $start" | bc)
rps=$(echo "scale=1; $REQUESTS / $elapsed" | bc)
echo "elapsed: ${elapsed}s   throughput: ${rps} req/s"
echo "final scraper metrics:"
curl -s "$TARGET/metrics"
