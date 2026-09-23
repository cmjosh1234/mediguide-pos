"""Internal, durable checkpoints. Cache failure must never approve or lose source data."""
import json
import structlog
from app.core.db import db_conn

log = structlog.get_logger()


class ArtifactCacheRepository:
    def get_many(self, kind: str, keys: list[str]) -> dict:
        result = {}
        try:
            with db_conn() as conn, conn.cursor() as cur:
                for offset in range(0, len(keys), 512):
                    cur.execute("SELECT cache_key,payload FROM ingestion_artifact_cache WHERE kind=%s AND cache_key=ANY(%s)", (kind, keys[offset:offset+512]))
                    result.update({row["cache_key"]: row["payload"] for row in cur.fetchall()})
        except Exception:
            log.warning("artifact_cache_read_unavailable", kind=kind)
        return result

    def put_many(self, kind: str, values: dict) -> None:
        if not values:
            return
        try:
            with db_conn() as conn, conn.cursor() as cur:
                cur.executemany("INSERT INTO ingestion_artifact_cache(cache_key,kind,payload) VALUES (%s,%s,%s::jsonb) ON CONFLICT(cache_key) DO UPDATE SET payload=EXCLUDED.payload", [(key, kind, json.dumps(value, allow_nan=False)) for key, value in values.items()])
                conn.commit()
        except Exception:
            log.warning("artifact_cache_checkpoint_unavailable", kind=kind)
