import threading
import time
from unittest.mock import patch

from app.worker import process_claimed_jobs


def test_claimed_jobs_process_concurrently_with_isolated_services():
    active = 0
    peak = 0
    lock = threading.Lock()
    seen = []

    class Service:
        def run_job(self, job_id, lease_guard=None):
            nonlocal active, peak
            with lock:
                active += 1
                peak = max(peak, active)
                seen.append(job_id)
            time.sleep(0.05)
            with lock:
                active -= 1

    jobs = [{"id": f"job-{i}", "attempt_count": 0} for i in range(4)]
    with patch("app.worker.IngestionService", side_effect=lambda: Service()):
        process_claimed_jobs(jobs, concurrency=2, worker_id="test-worker", heartbeat_seconds=60)

    assert sorted(seen) == [f"job-{i}" for i in range(4)]
    assert peak == 2


def test_each_concurrent_job_gets_a_distinct_service_instance():
    instances = []

    class Service:
        def __init__(self):
            instances.append(self)

        def run_job(self, _job_id, lease_guard=None):
            return None

    jobs = [{"id": "one"}, {"id": "two"}]
    with patch("app.worker.IngestionService", side_effect=lambda: Service()), patch(
        "app.worker.IngestionRepository.renew_lease", return_value=True
    ):
        process_claimed_jobs(jobs, concurrency=2, worker_id="test-worker", heartbeat_seconds=60)

    assert len(instances) == 2
    assert instances[0] is not instances[1]
