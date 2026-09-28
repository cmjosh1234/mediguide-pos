from __future__ import annotations

from concurrent import futures
from functools import lru_cache
import hmac
import time

import grpc
from grpc import StatusCode
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
import httpx
import structlog

from app.core.config import get_settings
from app.services.rag_service import RagService
from mediguide.aiworker.v1 import aiworker_pb2, aiworker_pb2_grpc

logger = structlog.get_logger(__name__)


@lru_cache(maxsize=1)
def get_rag_service() -> RagService:
    return RagService()


class AIWorkerServicer(aiworker_pb2_grpc.AIWorkerServiceServicer):
    def AskRAG(
        self,
        request: aiworker_pb2.AskRAGRequest,
        context: grpc.ServicerContext,
    ) -> aiworker_pb2.AskRAGResponse:
        started = time.monotonic()
        correlation_id = _metadata_value(context, "x-correlation-id")
        _authorize(context)
        if len(request.question.strip()) < 3:
            context.abort(StatusCode.INVALID_ARGUMENT, "question must be at least 3 characters")
        try:
            response = get_rag_service().ask(
                question=request.question,
                language=request.language or "en",
                program_area=request.program_area or None,
                country=request.country or None,
                top_k=request.top_k or None,
                history_summary=request.history_summary or None,
                recent_messages=[
                    {"role": message.role, "content": message.content}
                    for message in request.recent_messages
                ],
            )
            logger.info(
                "grpc_request",
                correlation_id=correlation_id,
                grpc_method="AskRAG",
                grpc_status="OK",
                duration_ms=round((time.monotonic() - started) * 1000, 2),
            )
            return _build_ask_response(response)
        except ValueError as exc:
            logger.warning(
                "grpc_request",
                correlation_id=correlation_id,
                grpc_method="AskRAG",
                grpc_status="INVALID_ARGUMENT",
                duration_ms=round((time.monotonic() - started) * 1000, 2),
            )
            context.abort(StatusCode.INVALID_ARGUMENT, str(exc))
        except (TimeoutError, httpx.TimeoutException):
            logger.warning(
                "grpc_request",
                correlation_id=correlation_id,
                grpc_method="AskRAG",
                grpc_status="DEADLINE_EXCEEDED",
                duration_ms=round((time.monotonic() - started) * 1000, 2),
            )
            context.abort(StatusCode.DEADLINE_EXCEEDED, "AI worker request timed out")
        except httpx.HTTPError:
            logger.warning(
                "grpc_request",
                correlation_id=correlation_id,
                grpc_method="AskRAG",
                grpc_status="UNAVAILABLE",
                duration_ms=round((time.monotonic() - started) * 1000, 2),
            )
            context.abort(StatusCode.UNAVAILABLE, "AI provider is temporarily unavailable")
        except RuntimeError:
            logger.exception(
                "grpc_request_failed_precondition",
                correlation_id=correlation_id,
                grpc_method="AskRAG",
                duration_ms=round((time.monotonic() - started) * 1000, 2),
            )
            context.abort(StatusCode.FAILED_PRECONDITION, "AI worker runtime is not configured")
        except Exception:
            logger.exception(
                "grpc_request_failed",
                correlation_id=correlation_id,
                grpc_method="AskRAG",
                duration_ms=round((time.monotonic() - started) * 1000, 2),
            )
            context.abort(StatusCode.INTERNAL, "AI worker request failed")

    def RunIngestionJob(
        self,
        request: aiworker_pb2.RunIngestionJobRequest,
        context: grpc.ServicerContext,
    ) -> aiworker_pb2.RunIngestionJobResponse:
        _authorize(context)
        context.abort(
            StatusCode.FAILED_PRECONDITION,
            "synchronous ingestion RPC is retired; ingestion_jobs are processed by ai-worker-loop",
        )


def build_grpc_server() -> grpc.Server:
    settings = get_settings()
    if _requires_worker_secret(settings.env) and not (settings.worker_api_secret or "").strip():
        raise RuntimeError("WORKER_API_SECRET is required outside development/test environments")
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
    aiworker_pb2_grpc.add_AIWorkerServiceServicer_to_server(AIWorkerServicer(), server)
    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
    health_servicer.set(
        "mediguide.aiworker.v1.AIWorkerService",
        health_pb2.HealthCheckResponse.SERVING,
    )
    server.add_insecure_port(f"{settings.grpc_host}:{settings.grpc_port}")
    return server


def _authorize(context: grpc.ServicerContext) -> None:
    secret = (get_settings().worker_api_secret or "").strip()
    if not secret:
        return
    provided = _metadata_value(context, "x-worker-secret")
    if not hmac.compare_digest(provided, secret):
        logger.warning(
            "grpc_auth_failed",
            correlation_id=_metadata_value(context, "x-correlation-id"),
            grpc_status="UNAUTHENTICATED",
        )
        context.abort(StatusCode.UNAUTHENTICATED, "invalid or missing x-worker-secret metadata")


def _metadata_value(context: grpc.ServicerContext, name: str) -> str:
    wanted = name.lower()
    for key, value in context.invocation_metadata():
        if key.lower() == wanted:
            return value
    return ""


def _requires_worker_secret(env: str) -> bool:
    return (env or "").strip().lower() not in {"", "development", "dev", "local", "test", "testing"}


def _build_ask_response(payload: dict) -> aiworker_pb2.AskRAGResponse:
    response = aiworker_pb2.AskRAGResponse(answer=str(payload.get("answer") or ""))
    for citation in payload.get("citations") or []:
        item = response.citations.add()
        item.chunk_id = str(_get_field(citation, "chunk_id") or "")
        item.title = str(_get_field(citation, "title") or "")
        item.country = str(_get_field(citation, "country") or "")
        item.source_name = str(_get_field(citation, "source_name") or "")
        item.source_version = str(_get_field(citation, "source_version") or "")
        item.page_start = int(_get_field(citation, "page_start") or 0)
        item.page_end = int(_get_field(citation, "page_end") or 0)
        item.similarity = float(_get_field(citation, "similarity") or 0)
    for chunk in payload.get("retrieved") or []:
        item = response.retrieved.add()
        item.id = str(_get_field(chunk, "id") or "")
        item.title = str(_get_field(chunk, "title") or "")
        item.content = str(_get_field(chunk, "content") or "")
        item.page_start = int(_get_field(chunk, "page_start") or 0)
        item.page_end = int(_get_field(chunk, "page_end") or 0)
        item.language = str(_get_field(chunk, "language") or "")
        item.program_area = str(_get_field(chunk, "program_area") or "")
        item.country = str(_get_field(chunk, "country") or "")
        item.source_name = str(_get_field(chunk, "source_name") or "")
        item.source_version = str(_get_field(chunk, "source_version") or "")
        item.similarity = float(_get_field(chunk, "similarity") or 0)
    safety = payload.get("safety") or {}
    response.safety.CopyFrom(
        aiworker_pb2.Safety(
            grounded=bool(safety.get("grounded")),
            provider=str(safety.get("provider") or ""),
            reason=str(safety.get("reason") or ""),
            standalone_question=str(safety.get("standalone_question") or ""),
            generation_error=str(safety.get("generation_error") or ""),
        )
    )
    return response


def _get_field(item, field: str):
    if isinstance(item, dict):
        return item.get(field)
    return getattr(item, field, None)
