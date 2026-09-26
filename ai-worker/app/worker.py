import os
import socket
import time
import uuid
from concurrent.futures import ThreadPoolExecutor, as_completed
from threading import Event, Thread
import structlog
from app.core.config import get_settings
from app.core.logging import configure_logging
from app.repositories.ingestion_repo import IngestionRepository
from app.services.ingestion_service import IngestionService

configure_logging()
log = structlog.get_logger()


def _default_worker_id() -> str:
    return f"{socket.gethostname()}:{os.getpid()}:{uuid.uuid4().hex[:8]}"


def _process_job(job: dict, worker_id: str, lease_seconds: int = 120, heartbeat_seconds: int = 30) -> None:
    job_id = str(job["id"])
    attempt = job.get("attempt_count") or 0
    stop = Event()
    lease_lost = Event()
    repo = IngestionRepository()

    def heartbeat() -> None:
        while not stop.wait(heartbeat_seconds):
            if not repo.renew_lease(job_id, worker_id, lease_seconds):
                lease_lost.set()
                log.warning("ingestion_lease_lost", job_id=job_id, worker_id=worker_id)
                return

    heartbeat_thread = Thread(target=heartbeat, name=f"lease-{job_id[:8]}", daemon=True)
    heartbeat_thread.start()
    try:
        log.info("processing_job", job_id=job_id, attempt=attempt + 1, worker_id=worker_id)
        # IngestionService carries per-job metrics/stage state, so each concurrent
        # job gets its own instance rather than sharing mutable service state.
        IngestionService().run_job(job_id, lease_guard=lambda: not lease_lost.is_set())
    except Exception as exc:
        log.exception(
            "job_failed",
            job_id=job_id,
            attempt=attempt + 1,
            worker_id=worker_id,
            error=str(exc),
        )
    finally:
        stop.set()
        heartbeat_thread.join(timeout=max(1, heartbeat_seconds))


def process_claimed_jobs(
    jobs: list[dict], concurrency: int, worker_id: str, lease_seconds: int = 120, heartbeat_seconds: int = 30
) -> None:
    if not jobs:
        return
    workers = max(1, min(concurrency, len(jobs)))
    with ThreadPoolExecutor(max_workers=workers, thread_name_prefix="ingestion") as executor:
        futures = [
            executor.submit(_process_job, job, worker_id, lease_seconds, heartbeat_seconds)
            for job in jobs
        ]
        for future in as_completed(futures):
            # _process_job logs ingestion failures itself. Calling result keeps
            # unexpected executor failures visible without serializing the batch.
            future.result()


def main() -> None:
    settings = get_settings()
    repo = IngestionRepository()
    worker_id = settings.worker_id.strip() or _default_worker_id()
    log.info(
        "worker_started",
        poll_interval=settings.worker_poll_interval_seconds,
        chunk_size=settings.chunk_size,
        chunk_overlap=settings.chunk_overlap,
        min_chunk_chars=settings.min_chunk_chars,
        embedding_request_batch_size=settings.embedding_request_batch_size,
        worker_concurrency=settings.worker_concurrency,
        worker_id=worker_id,
        ollama_base_url=settings.ollama_base_url,
        ollama_embedding_model=settings.ollama_embedding_model,
    )

    while settings.worker_enabled:
        # Never lease more documents than this process can start immediately.
        # Long-running jobs must not cause a queued-but-already-leased document
        # to expire before a thread becomes available for it.
        claim_limit = max(1, settings.worker_concurrency)
        recovered = repo.recover_expired_leases(limit=claim_limit * 4)
        if recovered:
            log.warning("expired_ingestion_leases_recovered", count=recovered, worker_id=worker_id)

        # Re-queue eligible failures first; the normal priority/aging claim then
        # considers them alongside newly queued documents without over-claiming.
        repo.claim_retryable_jobs(
            limit=max(settings.worker_batch_size, claim_limit),
            max_attempts=settings.worker_max_attempts,
            backoff_seconds=settings.worker_retry_backoff_seconds,
        )
        jobs = repo.claim_queued_jobs(
            claim_limit,
            worker_id=worker_id,
            lease_seconds=settings.worker_lease_seconds,
            priority_aging_seconds=settings.worker_priority_aging_seconds,
        )

        if not jobs:
            time.sleep(settings.worker_poll_interval_seconds)
            continue

        process_claimed_jobs(
            jobs,
            settings.worker_concurrency,
            worker_id,
            settings.worker_lease_seconds,
            settings.worker_heartbeat_seconds,
        )


if __name__ == "__main__":
    main()
