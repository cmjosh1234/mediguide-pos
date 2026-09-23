from datetime import datetime, timedelta, timezone
from unittest.mock import Mock

import pytest

from app.services.ingestion_service import IngestionService


@pytest.mark.parametrize("fails", [False, True])
def test_job_records_metrics_on_success_and_failure(fails):
    service = IngestionService.__new__(IngestionService)
    service.jobs = Mock()
    service.jobs.get_job.return_value = {"id": "job", "status": "queued", "created_at": datetime.now(timezone.utc) - timedelta(seconds=5)}
    service._process = Mock(side_effect=ValueError("test failure") if fails else None)
    if fails:
        with pytest.raises(ValueError):
            service.run_job("job")
        service.jobs.mark_failed.assert_called_once()
    else:
        service.run_job("job")
        service.jobs.mark_completed.assert_called_once()
    saved_job, metrics = service.jobs.record_metrics.call_args.args
    assert saved_job == "job"
    assert metrics["processing_seconds"] >= 0
    assert metrics["queue_seconds"] >= 5


def test_metrics_write_failure_does_not_fail_successful_processing():
    service = IngestionService.__new__(IngestionService)
    service.jobs = Mock()
    service.jobs.get_job.return_value = {"id": "job", "status": "queued"}
    service.jobs.record_metrics.side_effect = RuntimeError("metrics unavailable")
    service._process = Mock()
    service.run_job("job")
    service.jobs.mark_completed.assert_called_once()
    service.jobs.mark_failed.assert_not_called()
