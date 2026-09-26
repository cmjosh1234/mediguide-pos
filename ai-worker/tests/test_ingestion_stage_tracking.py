from unittest.mock import Mock

import pytest

from app.services.ingestion_service import IngestionService


def test_failed_processing_marks_active_stage_failed_before_job_failure():
    service = IngestionService.__new__(IngestionService)
    service.jobs = Mock()
    service.settings = Mock(worker_retry_backoff_seconds=30)
    service.jobs.get_job.return_value = {"id": "job", "status": "queued"}
    service._process = Mock(side_effect=RuntimeError("embedding provider unavailable"))

    with pytest.raises(RuntimeError):
        service.run_job("job")

    service.jobs.fail_active_stage.assert_called_once_with("job", "embedding provider unavailable")
    service.jobs.mark_failed.assert_called_once()


def test_successful_processing_does_not_mark_stage_failed():
    service = IngestionService.__new__(IngestionService)
    service.jobs = Mock()
    service.settings = Mock(worker_retry_backoff_seconds=30)
    service.jobs.get_job.return_value = {"id": "job", "status": "queued"}
    service._process = Mock()

    service.run_job("job")

    service.jobs.fail_active_stage.assert_not_called()
    service.jobs.mark_completed.assert_called_once_with("job")
