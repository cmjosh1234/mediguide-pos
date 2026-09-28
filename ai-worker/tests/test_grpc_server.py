from types import SimpleNamespace

from grpc_health.v1 import health_pb2

from app import grpc_server


def test_build_grpc_server_registers_standard_health_service(monkeypatch):
    class FakeServer:
        def __init__(self):
            self.addresses = []

        def add_insecure_port(self, address):
            self.addresses.append(address)

    class FakeHealthServicer:
        def __init__(self):
            self.statuses = {}

        def set(self, service, status):
            self.statuses[service] = status

    server = FakeServer()
    registered = {}

    monkeypatch.setattr(
        grpc_server,
        "get_settings",
        lambda: SimpleNamespace(
            env="test",
            worker_api_secret="",
            grpc_host="127.0.0.1",
            grpc_port=50051,
        ),
    )
    monkeypatch.setattr(grpc_server.grpc, "server", lambda _executor: server)
    monkeypatch.setattr(
        grpc_server.aiworker_pb2_grpc,
        "add_AIWorkerServiceServicer_to_server",
        lambda servicer, target: registered.update(ai=(servicer, target)),
    )
    monkeypatch.setattr(grpc_server.health, "HealthServicer", FakeHealthServicer)
    monkeypatch.setattr(
        grpc_server.health_pb2_grpc,
        "add_HealthServicer_to_server",
        lambda servicer, target: registered.update(health=(servicer, target)),
    )

    built = grpc_server.build_grpc_server()

    assert built is server
    assert registered["ai"][1] is server
    assert registered["health"][1] is server
    assert registered["health"][0].statuses == {
        "": health_pb2.HealthCheckResponse.SERVING,
        "mediguide.aiworker.v1.AIWorkerService": health_pb2.HealthCheckResponse.SERVING,
    }
    assert server.addresses == ["127.0.0.1:50051"]
