FROM python:3.11-slim AS base

WORKDIR /app

# Install deps first for layer caching.
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app ./app

EXPOSE 8082
# Single worker is fine; scoring is CPU-light and stateless. Scale via replicas.
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8082"]
