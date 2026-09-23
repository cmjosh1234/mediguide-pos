# Document upload performance: phases 1–4

## Scope and safety

This workflow applies to guideline-version PDF and UTF-8 Markdown source uploads.
It does not replace image, managed-document, or outbreak attachment uploads.
It never approves blocks, figures, or a regenerated projection, and never publishes
a guideline. Processing completion means **ready for editorial review**.

## Phase 1: measurement

### UCG baseline measured on 22 September 2026

Read-only isolated Linux worker container, 2 CPUs, 4 GiB memory cap, network
disabled, current extractor. Source: `UCG2023.pdf`, 13,527,788 bytes, SHA-256
`d7814fcc7f0b0575a54bcdaeae9e1c88d04122c2732d6a0a9ff1d181ae456f02`.

| Measurement | First successful run |
|---|---:|
| Pages | 1,161 |
| Extraction | 79.628 s |
| Table extraction (within extraction) | 64.818 s |
| OCR (within extraction; 9 attempts) | 3.605 s |
| Chunking | 0.064 s |
| Sections / blocks / chunks | 2,481 / 6,815 / 6,835 |
| Tables / assets | 856 / 36 |

An earlier attempt under a 2 GiB cap exited with code 137 without a completed
report. The 4 GiB run completed. This is a resource-sizing warning, not evidence
of an upload-network failure. Table extraction consumed approximately 81% of
extraction time in this run. Extracted counts are diagnostic, not evidence that
the hierarchy, clinical tables or figures have passed review. This is an initial
extraction baseline, **not** a full-chain upload-to-index performance claim.

Each ingestion job now records `metrics` accessible through the authenticated
`GET /api/v2/guideline-versions/{version}/upload-jobs/{job}` endpoint:

- Upload-session wall time (including pauses), verification/queue submission time.
- Queue wait, total processing time, and time per processing stage.
- PDF table extraction and OCR time/attempt counts (nested within parsing time).
- Source bytes, pages, sections, blocks, tables, figures/assets and chunks.
- A checksum-no-op marker, so repeat uploads are not mistaken for fast full extraction.

The worker also emits `ingestion_benchmark` JSON log events. Metrics contain no
document body or credentials. Save the API JSON responses for comparison:

```sh
python3 scripts/ingestion-benchmark-report.py ucg-run-1.json ucg-run-2.json ucg-run-3.json
```

For extraction-only measurement, run inside the worker environment:

```sh
PYTHONPATH=. python scripts/benchmark_extraction.py /path/to/UCG2023.pdf
```

Benchmark UCG first, then a small text PDF, a scanned PDF and Markdown. Use three
fresh draft versions for full runs; keep the source checksum, worker CPU/memory,
embedding provider/model, network profile and concurrency fixed. Compare medians,
not one unusually fast run. Measure browser upload time separately (DevTools,
including slow-network and interruption tests). Session wall time includes idle
pauses and is not a bandwidth measurement. OCR/table timings are **subsets** of
parsing and must not be added again to total processing time.

The CLI extraction benchmark is not a measurement of upload, queue, storage,
embedding-provider latency or database persistence. Full-chain results require
the instrumented backend/worker and a draft upload through the dashboard.

## Phase 2: storage and backend

Migration `00054_guideline_upload_sessions.sql` adds durable upload sessions and
job metrics. Apply migrations before starting the updated backend and worker.

New endpoints require `guideline.markdown.upload`:

| Endpoint under `/api/v2/guideline-versions/{version}` | Purpose |
|---|---|
| `GET /upload-capabilities` | Discover direct-upload availability and size limit |
| `POST /uploads` | Begin a session with filename, byte size and SHA-256 checksum |
| `GET /uploads` | List the requesting user's recent sessions |
| `GET /uploads/{session}` | Recover stored parts and attached job |
| `POST /uploads/{session}/parts/{number}` | Obtain a short-lived signed PUT URL |
| `POST /uploads/{session}/complete` | Verify bytes and queue exactly one job for this session |
| `DELETE /uploads/{session}` | Abort an unattached source upload |
| `GET /upload-jobs/{job}` | Read processing state and metrics |

Sessions are scoped to user and version, expire after 24 hours, and use 8 MiB
parts. Signed URLs expire after 15 minutes and are refreshed on retry. No storage
credentials reach the browser. Completion checks stored sizes/order, SHA-256,
PDF signature or Markdown UTF-8 validity, and version mutability. Database row
locking makes repeated completion return the same job. A stored object survives
a crash between MinIO completion and database commit and can be reconciled.

Every 30 seconds, backend maintenance aborts expired unattached sessions and
creates deduplicated in-app success/failure notifications for terminal jobs.
Completed/attached sources are never deleted by this cleanup. Pause is not cancel.
The existing backend multipart upload endpoint remains available.

### Enable direct uploads

Keep `GUIDELINE_DIRECT_UPLOADS=false` until the following are verified:

1. The browser can reach the **S3 API** hostname, not the MinIO console or Docker-only hostname.
2. The storage proxy preserves the signed host, path and query string, supports PUT
   and OPTIONS, and allows the configured part sizes and transfer duration.
3. HTTPS is used for storage when the dashboard uses HTTPS.
4. CORS permits only the actual dashboard origins. Do not expose the bucket publicly.

Local configuration (backend/container environment):

```dotenv
GUIDELINE_DIRECT_UPLOADS=true
S3_PUBLIC_ENDPOINT=localhost:9000
S3_PUBLIC_SSL=false
MINIO_API_CORS_ALLOW_ORIGIN=http://localhost:3000
```

Production example, replacing the hostname/origin with your actual configuration:

```dotenv
GUIDELINE_DIRECT_UPLOADS=true
S3_PUBLIC_ENDPOINT=storage.example.org
S3_PUBLIC_SSL=true
MINIO_API_CORS_ALLOW_ORIGIN=https://dashboard.example.org
```

`S3_PUBLIC_ENDPOINT` has no scheme or path. Internal `S3_ENDPOINT` remains unchanged.
The signer currently uses the default MinIO `us-east-1` region. A custom-region
S3 deployment needs matching signer configuration before enablement.
Restart/recreate MinIO and the backend after changing their environment. Do not
enable direct uploads if CORS/preflight or signed PUT smoke tests fail.

Rollback: disable the flag and use standard uploads. Leave the additive migration
in place so metrics and previous session records are preserved. Resume an old
direct-upload session after re-enabling the feature, within its expiry window.

## Phase 3: editor workflow

Open a draft version's Upload Source dialog and select a PDF or Markdown file.
The Create Guideline wizard uses the same uploader; its URL retains the created
document/version IDs so refreshing does not create another guideline or version.
The client hashes the file, sends up to three parts concurrently, retries failed
parts up to three attempts, and displays byte percentage and transfer rate.
Hashing uses the browser's Web Crypto API and currently reads the selected file
into memory; the server's configured size cap still applies.

- **Pause:** stops active transfers; completed parts remain in MinIO.
- **Resume:** reselect the same file after refresh; checksum/size must match.
- **Cancel upload:** aborts and removes the unattached upload; cannot delete a queued source.
- **Processing:** shows server stages separately from upload progress. Closing the
  dialog does not stop processing. Reopen it to restore the job.
- **Ready for review:** open Editorial Review and follow the normal approval flow.

Only session/job identifiers are stored locally, scoped by account and version.
File bytes and signed URLs are not saved in localStorage. The server also lists
recent unfinished uploads if browser state is lost. Standard-upload fallback has
an indeterminate upload indicator rather than resumable multipart controls.

## Verification checklist

- Backend tests: ownership, bounds, missing parts, checksum, published immutability,
  completed-object recovery, one-job completion, Markdown revision and expiry cleanup.
- Dashboard tests: completed parts skipped, recovery after lost completion response,
  wrong-file rejection, pause before session creation; typecheck and lint.
- Worker extraction/ingestion regression tests and UCG extraction-only baseline.
- Local MinIO smoke test (test credentials only):

```sh
cd backend
MINIO_TEST_ENDPOINT=localhost:9000 go test ./internal/storage -run TestMinioMultipartIntegration -v
```

This creates/removes one uniquely named `upload-tests` object, not guideline data.
Before production enablement, test authenticated browser CORS, network interruption,
refresh during transfer/verification, completed/failed notifications, and a full
UCG draft upload. These deployment checks are distinct from unit-test completion.

## Phase 4: reusable computation and retry checkpoints

Apply migration `00055_ingestion_artifact_cache.sql` before enabling the updated
worker. Reuse is independent of direct uploads and works with both upload paths.
It is disabled by default in the worker and all tracked deployment templates.

```dotenv
INGESTION_ARTIFACT_REUSE=true
ARTIFACT_CACHE_EPOCH=1
EMBEDDING_CACHE_MODEL_REVISION=
```

Recreate both worker services after changing configuration. Start locally; do not
enable production reuse until the provider identity and deployment smoke tests
have been verified. Set `INGESTION_ARTIFACT_REUSE=false` to return to the existing
processing path. No cache cleanup is required for rollback.

| Artifact | Identity and storage |
|---|---|
| Raw extraction | Source SHA-256, source format, extractor source/dependency versions, OCR engine/trained-data/settings, epoch; Markdown also includes fallback title. Compressed JSON in private MinIO `ingestion-cache/v1/extraction/`, with a checksum marker in PostgreSQL. |
| OCR | Rendered page PNG SHA-256, Tesseract version, English trained-data checksum, render scale/language/arguments, extraction wrapper revision and epoch. Text checkpoint in PostgreSQL. |
| Embedding | Exact input string SHA-256 plus provider, model, immutable revision, dimensions, endpoint, provider implementation/dependencies and epoch. Vector checkpoint in PostgreSQL. |

Ollama resolves mutable model tags to their installed digest for each job. For
OpenAI or sentence-transformers, set `EMBEDDING_CACHE_MODEL_REVISION` to an
operator-maintained immutable deployment/weights revision and change it whenever
model behavior changes. If a trustworthy identity cannot be established, that
cache is bypassed. Never leave a fixed revision across model upgrades. Restart
workers after changing OCR binaries or trained data. Increment the epoch to
invalidate all artifact classes without deleting active document content.

Successful embedding batches are committed before the next provider call. Failed
jobs can therefore reuse earlier batches. Duplicate identical inputs within a
job are computed once and restored in their original order. Invalid vector
dimensions, non-finite numbers, and mismatched response counts fail the job;
Ollama's dynamic shortened-input fallback is never checkpointed under the
original exact-input identity.

Extraction is cached **before** adding version-owned storage keys, source PDF
attachments, revision IDs or job IDs. Each target still gets its own projection,
assets and existing editorial workflow. Cache hits never approve or publish
anything. Source-supersession checks, cancellation checks and transactional final
projection replacement remain in place. A partially successful best-effort
table/OCR/image extraction is not saved as a complete extraction artifact.

Missing/corrupt blobs or unavailable cache storage cause recomputation, not loss
of source content. Blob names include their payload checksum so concurrent cache
writers cannot overwrite a different payload under a valid marker. Cache access
is internal only; do not expose its MinIO prefix or add public cache lookup APIs.

### Metrics, retention and repeatable verification

Job metrics include extraction/OCR hits and misses, embedding hits, unique input
misses, provider calls and whether a complete extraction checkpoint was withheld.
Cached extraction metadata retains the original extraction timings; use the
current job's parsing stage time for warm-run latency, not those original timings.

Caches are disposable but currently have no automatic retention policy. Monitor
PostgreSQL size and the private MinIO prefix. Any retention/erasure policy must
cover both stores: OCR checkpoints contain source text and extracted artifacts
contain source content. Deleting a guideline does not automatically erase shared
cache artifacts. Eviction must never target the separate `guidelines/` source and
published-asset prefixes. Missing evicted artifacts safely recompute.

Run the isolated acceptance benchmark in the worker environment with a 4 GiB
memory limit for UCG:

```sh
PYTHONPATH=. python scripts/benchmark_artifact_reuse.py /path/to/UCG2023.pdf
python -m pytest tests/test_ingestion_artifacts.py -q
```

The benchmark uses temporary durable SQLite/filesystem adapters and deterministic
development embeddings, not production PostgreSQL/MinIO or a clinical embedding
model. It verifies cold versus identical extraction, byte-equivalent serialized
output, unchanged embedding reuse, and one changed embedding input. The changed
input scenario is **not** a revised-PDF end-to-end upload benchmark. The script
does not change clinical versions, reviews or publication state. Full-chain
staging tests and production-provider timing remain phase 6 rollout work.

### UCG artifact benchmark: 23 September 2026 (local time)

Isolated Linux container, 2 CPUs, 4 GiB cap, no network; same UCG checksum as the
baseline above. One cold/warm pair, not a median or tail-latency claim:

| Scenario | Extraction | Embedding calls | Cached chunk inputs |
|---|---:|---:|---:|
| Cold | 69.987 s | 79 batches (64 unique inputs maximum per batch) | 0 |
| Identical | 0.236 s | 0 | 6,835 |
| One changed embedding input | 0.248 s | 1 | 6,834 |

There were 5,029 unique cold embedding inputs among 6,835 chunks. All runs retained
1,161 pages, 2,481 sections, 6,815 blocks, 856 tables and 36 extracted assets. The
serialized extraction digest matched across all three scenarios:
`be1625a8bb92fccb097dc93ca145e6b19fecab686608b5d2bc6741627fa81239`.
Cold OCR produced nine checkpoints. Peak process RSS was 3,582,944 KiB (about
3.42 GiB); caching does not solve cold-run extraction memory use. Measured
development-embedding times were 0.690 / 0.162 / 4.870 seconds respectively;
the last single-run result illustrates why provider speed claims require repeated
measurements rather than inference from cache hit counts.
