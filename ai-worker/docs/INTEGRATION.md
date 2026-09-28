# Integration Guide

## 1. Start backend infrastructure

```bash
cd backend
cp .env.example .env
docker compose up -d postgres minio redis
make migrate-up
make seed
```

## 2. Start the AI worker service

```bash
cd ../ai-worker
cp .env.example .env
ollama pull mxbai-embed-large:latest
uvicorn app.main:app --reload --port 8090
```

The FastAPI listener exposes health/readiness and extraction preview endpoints.
Backend-to-worker RAG traffic uses the internal gRPC listener on port `50051`.

## 3. Start worker loop

```bash
python -m app.worker
```

## 4. Upload a guideline PDF through the Go backend

The Go backend creates a `guideline_versions` row and an `ingestion_jobs` row.

## 5. AI worker processes the job

The worker downloads the PDF from MinIO, extracts HTML/Markdown, chunks it, embeds chunks, and stores them in PostgreSQL.

## 6. Ask through the Go backend

```bash
curl -X POST http://localhost:8080/api/public/assistant/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"How is severe malaria managed?","program_area":"Malaria","language":"en"}'
```

The Go backend keeps a persistent gRPC connection to the AI worker, applies
deadlines and bounded retries to safe RAG calls, and propagates `X-Request-ID`
as `x-correlation-id` metadata. Ingestion is not dispatched over gRPC: the
durable `ingestion_jobs` queue is claimed directly by `python -m app.worker`.

## Production note

The backend and AI worker now default to Ollama `mxbai-embed-large:latest` with `EMBEDDING_DIM=1024`. If you change embedding models, migrate the `guideline_chunks.embedding` column to the new vector size before ingesting documents.
