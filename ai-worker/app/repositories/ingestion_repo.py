from __future__ import annotations

from typing import Any
import json

from app.core.db import db_conn


class IngestionRepository:
    def start_stage(self, job_id: str, stage: str, percent: int) -> None:
        value = max(0, min(100, percent))
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                """
                UPDATE ingestion_tasks
                SET status='completed', progress_percent=100, completed_at=now(), updated_at=now()
                WHERE job_id=%s AND status='running' AND stage<>%s
                """,
                (job_id, stage),
            )
            cur.execute(
                """
                INSERT INTO ingestion_tasks (job_id, stage, status, progress_percent, attempt_count, started_at)
                VALUES (%s, %s, 'running', %s, 1, now())
                ON CONFLICT (job_id, stage) DO UPDATE
                SET status='running',
                    progress_percent=EXCLUDED.progress_percent,
                    error=NULL,
                    started_at=CASE
                      WHEN ingestion_tasks.status='running' THEN ingestion_tasks.started_at
                      ELSE now()
                    END,
                    completed_at=NULL,
                    attempt_count=CASE
                      WHEN ingestion_tasks.status='running' THEN ingestion_tasks.attempt_count
                      ELSE ingestion_tasks.attempt_count + 1
                    END,
                    updated_at=now()
                """,
                (job_id, stage, value),
            )
            cur.execute(
                "UPDATE ingestion_jobs SET progress_stage=%s, progress_percent=%s, updated_at=now() WHERE id=%s AND status='running'",
                (stage, value, job_id),
            )
            conn.commit()

    def fail_active_stage(self, job_id: str, error: str) -> None:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                """
                UPDATE ingestion_tasks
                SET status='failed', error=%s, completed_at=now(), updated_at=now()
                WHERE job_id=%s AND status='running'
                """,
                (error[:4000], job_id),
            )
            conn.commit()

    def list_stages(self, job_id: str) -> list[dict[str, Any]]:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                "SELECT stage, status, progress_percent, attempt_count, error, started_at, completed_at FROM ingestion_tasks WHERE job_id=%s ORDER BY created_at ASC, id ASC",
                (job_id,),
            )
            return cur.fetchall()
    def record_metrics(self, job_id: str, metrics: dict) -> None:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute("UPDATE ingestion_jobs SET metrics_json=COALESCE(metrics_json, '{}'::jsonb) || %s::jsonb WHERE id=%s", (json.dumps(metrics), job_id))
            conn.commit()

    def _has_attempt_count(self) -> bool:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                """
                SELECT EXISTS (
                    SELECT 1
                    FROM information_schema.columns
                    WHERE table_name = 'ingestion_jobs'
                      AND column_name = 'attempt_count'
                ) AS present
                """
            )
            row = cur.fetchone()
            return bool(row and row.get("present"))

    def claim_queued_jobs(
        self,
        limit: int = 1,
        worker_id: str = "",
        lease_seconds: int = 120,
        priority_aging_seconds: int = 900,
    ) -> list[dict[str, Any]]:
        """Atomically pick up queued jobs and mark them running."""
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                """
                WITH picked AS (
                    SELECT id
                    FROM ingestion_jobs
                    WHERE status = 'queued'
                      AND job_type IN ('pdf_ingestion', 'markdown_ingestion', 'original_text_index')
                      AND deleted_at IS NULL
                      AND (next_attempt_at IS NULL OR next_attempt_at <= now())
                    ORDER BY
                      (
                        priority
                        + LEAST(
                            50,
                            FLOOR(EXTRACT(EPOCH FROM (now() - created_at)) / %s)::int
                          )
                      ) DESC,
                      priority DESC,
                      created_at ASC,
                      id ASC
                    LIMIT %s
                    FOR UPDATE SKIP LOCKED
                )
                UPDATE ingestion_jobs j
                SET status = 'running', progress_stage='downloading', progress_percent=5,
                    worker_id = NULLIF(%s, ''), claimed_at = now(), heartbeat_at = now(),
                    lease_expires_at = now() + (%s * interval '1 second'),
                    next_attempt_at = NULL,
                    started_at = coalesce(started_at, now()), updated_at = now()
                FROM picked
                WHERE j.id = picked.id
                RETURNING j.*
                """,
                (max(60, priority_aging_seconds), limit, worker_id, max(30, lease_seconds)),
            )
            rows = cur.fetchall()
            conn.commit()
            return rows

    def claim_retryable_jobs(
        self, limit: int = 1, max_attempts: int = 3, backoff_seconds: int = 30
    ) -> list[dict[str, Any]]:
        """Pick up previously-failed jobs that are within the retry limit and past their back-off window."""
        if not self._has_attempt_count():
            return []
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                """
                WITH picked AS (
                    SELECT id
                    FROM ingestion_jobs
                    WHERE status = 'failed'
                      AND job_type IN ('pdf_ingestion', 'markdown_ingestion', 'original_text_index')
                      AND deleted_at IS NULL
                      AND coalesce(attempt_count, 0) < %s
                      AND COALESCE(
                        next_attempt_at,
                        completed_at + (coalesce(attempt_count, 1) * %s * interval '1 second'),
                        now()
                      ) <= now()
                    ORDER BY completed_at ASC NULLS FIRST
                    LIMIT %s
                    FOR UPDATE SKIP LOCKED
                )
                UPDATE ingestion_jobs j
                SET status = 'queued', error = NULL, worker_id=NULL,
                    heartbeat_at=NULL, lease_expires_at=NULL, updated_at = now()
                FROM picked
                WHERE j.id = picked.id
                RETURNING j.*
                """,
                (max_attempts, max(1, backoff_seconds), limit),
            )
            rows = cur.fetchall()
            conn.commit()
            return rows

    def recover_expired_leases(self, limit: int = 100) -> int:
        """Return abandoned running jobs to the queue after their worker lease expires."""
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                """
                WITH expired AS (
                    SELECT id
                    FROM ingestion_jobs
                    WHERE status = 'running'
                      AND deleted_at IS NULL
                      AND lease_expires_at IS NOT NULL
                      AND lease_expires_at < now()
                    ORDER BY lease_expires_at ASC
                    LIMIT %s
                    FOR UPDATE SKIP LOCKED
                )
                UPDATE ingestion_jobs j
                SET status='queued', progress_stage='queued',
                    worker_id=NULL, heartbeat_at=NULL, lease_expires_at=NULL,
                    updated_at=now()
                FROM expired
                WHERE j.id=expired.id
                RETURNING j.id
                """,
                (limit,),
            )
            rows = cur.fetchall()
            for row in rows:
                cur.execute(
                    """
                    UPDATE ingestion_tasks
                    SET status='failed',
                        error=COALESCE(error, 'worker lease expired'),
                        completed_at=COALESCE(completed_at, now()),
                        updated_at=now()
                    WHERE job_id=%s AND status='running'
                    """,
                    (row["id"],),
                )
            conn.commit()
            return len(rows)

    def renew_lease(self, job_id: str, worker_id: str, lease_seconds: int) -> bool:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                """
                UPDATE ingestion_jobs
                SET heartbeat_at=now(),
                    lease_expires_at=now() + (%s * interval '1 second'),
                    updated_at=now()
                WHERE id=%s AND status='running' AND worker_id=%s
                """,
                (max(30, lease_seconds), job_id, worker_id),
            )
            renewed = cur.rowcount == 1
            conn.commit()
            return renewed

    def get_job(self, job_id: str) -> dict[str, Any] | None:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute("SELECT * FROM ingestion_jobs WHERE id = %s", (job_id,))
            return cur.fetchone()

    def mark_running(self, job_id: str) -> None:
        """Safety guard: transitions a queued job to running if not already running.
        The worker loop already handles this atomically; this method is kept for
        API-triggered jobs (the /run endpoint) where the claim step is skipped."""
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                "UPDATE ingestion_jobs SET status='running', progress_stage='downloading', progress_percent=5, started_at=coalesce(started_at, now()), updated_at=now() WHERE id=%s AND status='queued'",
                (job_id,),
            )
            cur.execute(
                """
                UPDATE guideline_markdown_revisions
                SET structured_content_status='processing', updated_at=now()
                WHERE regeneration_job_id=%s AND deleted_at IS NULL
                """,
                (job_id,),
            )
            cur.execute(
                """
                UPDATE guideline_versions gv
                SET structured_content_status='processing', updated_at=now()
                FROM guideline_markdown_revisions revision
                WHERE revision.regeneration_job_id=%s
                  AND revision.id=gv.current_markdown_revision_id
                  AND revision.deleted_at IS NULL
                """,
                (job_id,),
            )
            conn.commit()

    def mark_completed(self, job_id: str) -> None:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                "UPDATE ingestion_tasks SET status='completed', progress_percent=100, completed_at=now(), updated_at=now() WHERE job_id=%s AND status='running'",
                (job_id,),
            )
            cur.execute(
                "UPDATE ingestion_jobs SET status='completed', progress_stage='completed', progress_percent=100, completed_at=now(), updated_at=now(), error=NULL, heartbeat_at=NULL, lease_expires_at=NULL WHERE id=%s AND status='running'",
                (job_id,),
            )
            conn.commit()

    def set_progress(self, job_id: str, stage: str, percent: int) -> None:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                "UPDATE ingestion_jobs SET progress_stage=%s, progress_percent=%s, updated_at=now() WHERE id=%s AND status='running'",
                (stage, max(0, min(100, percent)), job_id),
            )
            conn.commit()

    def cancellation_requested(self, job_id: str) -> bool:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                "SELECT status='cancel_requested' AS requested FROM ingestion_jobs WHERE id=%s",
                (job_id,),
            )
            row = cur.fetchone()
            return bool(row and row.get("requested"))

    def mark_canceled(self, job_id: str) -> None:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                "UPDATE ingestion_tasks SET status='canceled', completed_at=now(), updated_at=now() WHERE job_id=%s AND status='running'",
                (job_id,),
            )
            cur.execute(
                "UPDATE ingestion_jobs SET status='canceled', progress_stage='canceled', canceled_at=now(), completed_at=now(), updated_at=now(), heartbeat_at=NULL, lease_expires_at=NULL WHERE id=%s AND status='cancel_requested'",
                (job_id,),
            )
            cur.execute(
                "UPDATE guideline_markdown_revisions SET structured_content_status='canceled', review_state='draft', updated_at=now() WHERE regeneration_job_id=%s",
                (job_id,),
            )
            cur.execute(
                "UPDATE guideline_versions gv SET structured_content_status='canceled', updated_at=now() FROM guideline_markdown_revisions r WHERE r.regeneration_job_id=%s AND gv.current_markdown_revision_id=r.id",
                (job_id,),
            )
            conn.commit()

    def mark_superseded(self, job_id: str) -> None:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                "UPDATE ingestion_tasks SET status='canceled', completed_at=now(), updated_at=now() WHERE job_id=%s AND status='running'",
                (job_id,),
            )
            cur.execute(
                "UPDATE ingestion_jobs SET status='canceled', progress_stage='superseded', canceled_at=now(), completed_at=now(), updated_at=now(), heartbeat_at=NULL, lease_expires_at=NULL WHERE id=%s AND status='running'",
                (job_id,),
            )
            cur.execute(
                """
                UPDATE guideline_markdown_revisions revision
                SET structured_content_status='canceled', review_state='draft', updated_at=now()
                WHERE revision.regeneration_job_id=%s
                  AND NOT EXISTS (
                    SELECT 1 FROM guideline_versions gv
                    WHERE gv.current_markdown_revision_id=revision.id
                      AND gv.deleted_at IS NULL
                  )
                """,
                (job_id,),
            )
            # A superseded job must never leave the current revision stuck in
            # processing. Restore the state of its last accepted projection.
            cur.execute(
                """
                UPDATE guideline_markdown_revisions revision
                SET structured_content_status=CASE
                      WHEN gv.structured_markdown_revision_id=revision.id
                        THEN 'review_required'
                      ELSE 'outdated'
                    END,
                    review_state=CASE
                      WHEN gv.structured_markdown_revision_id=revision.id
                        THEN 'review_required'
                      ELSE 'draft'
                    END,
                    updated_at=now()
                FROM guideline_versions gv
                WHERE revision.regeneration_job_id=%s
                  AND gv.current_markdown_revision_id=revision.id
                  AND revision.deleted_at IS NULL
                  AND gv.deleted_at IS NULL
                """,
                (job_id,),
            )
            cur.execute(
                """
                UPDATE guideline_versions gv
                SET structured_content_status=CASE
                      WHEN gv.structured_markdown_revision_id=revision.id
                        THEN 'review_required'
                      ELSE 'outdated'
                    END,
                    status=CASE
                      WHEN gv.structured_markdown_revision_id=revision.id
                        THEN 'review_required'
                      ELSE gv.status
                    END,
                    updated_at=now()
                FROM guideline_markdown_revisions revision
                WHERE revision.regeneration_job_id=%s
                  AND gv.current_markdown_revision_id=revision.id
                  AND revision.deleted_at IS NULL
                  AND gv.deleted_at IS NULL
                """,
                (job_id,),
            )
            conn.commit()

    def complete_noop_comparison(self, job_id: str) -> None:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                "UPDATE guideline_regeneration_reviews SET after_snapshot=before_snapshot, comparison='{\"no_changes\":true}'::jsonb, updated_at=now() WHERE job_id=%s AND deleted_at IS NULL",
                (job_id,),
            )
            cur.execute(
                """
                UPDATE guideline_markdown_revisions revision
                SET structured_content_status='review_required',
                    review_state='review_required',
                    updated_at=now()
                FROM guideline_versions gv
                WHERE revision.regeneration_job_id=%s
                  AND gv.current_markdown_revision_id=revision.id
                  AND revision.deleted_at IS NULL
                  AND gv.deleted_at IS NULL
                """,
                (job_id,),
            )
            cur.execute(
                """
                UPDATE guideline_versions gv
                SET structured_markdown_revision_id=revision.id,
                    structured_content_status='review_required',
                    status='review_required',
                    updated_at=now()
                FROM guideline_markdown_revisions revision
                WHERE revision.regeneration_job_id=%s
                  AND gv.current_markdown_revision_id=revision.id
                  AND revision.deleted_at IS NULL
                  AND gv.deleted_at IS NULL
                """,
                (job_id,),
            )
            conn.commit()

    def mark_failed(self, job_id: str, error: str, retry_backoff_seconds: int = 30) -> None:
        """Increment attempt_count and mark job failed."""
        if not self._has_attempt_count():
            with db_conn() as conn, conn.cursor() as cur:
                cur.execute(
                    """UPDATE ingestion_jobs
                       SET status='failed',
                           error=%s,
                           completed_at=now(),
                           updated_at=now(),
                           heartbeat_at=NULL,
                           lease_expires_at=NULL
                       WHERE id=%s""",
                    (error[:4000], job_id),
                )
                self._mark_revision_failed(cur, job_id)
                conn.commit()
            return
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute(
                """UPDATE ingestion_jobs
                   SET status='failed',
                       error=%s,
                       completed_at=now(),
                       updated_at=now(),
                       attempt_count=coalesce(attempt_count, 0) + 1,
                       next_attempt_at=now() + ((coalesce(attempt_count, 0) + 1) * %s * interval '1 second'),
                       heartbeat_at=NULL,
                       lease_expires_at=NULL
                   WHERE id=%s""",
                (error[:4000], max(1, retry_backoff_seconds), job_id),
            )
            self._mark_revision_failed(cur, job_id)
            conn.commit()

    @staticmethod
    def _mark_revision_failed(cur, job_id: str) -> None:
        cur.execute(
            """
            UPDATE guideline_markdown_revisions
            SET structured_content_status='failed', updated_at=now()
            WHERE regeneration_job_id=%s AND deleted_at IS NULL
            """,
            (job_id,),
        )
        cur.execute(
            """
            UPDATE guideline_versions gv
            SET structured_content_status='failed', updated_at=now()
            FROM guideline_markdown_revisions revision
            WHERE revision.regeneration_job_id=%s
              AND revision.id=gv.current_markdown_revision_id
              AND revision.deleted_at IS NULL
            """,
            (job_id,),
        )
