from __future__ import annotations

import contextlib
import os
import socket
import subprocess
import time
from pathlib import Path
from typing import Iterator

import pytest

from cmdagent_client import Client
from cmdagent_client.gen.agent.v1 import agent_pb2


def _find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def _wait_for_daemon(client: Client, timeout: float = 10.0) -> None:
    deadline = time.time() + timeout
    last_error: Exception | None = None
    while time.time() < deadline:
        try:
            client.list_executions()
            return
        except Exception as exc:  # pragma: no cover - startup polling
            last_error = exc
            time.sleep(0.1)
    raise RuntimeError(f"cmdagentd did not become ready: {last_error}")


def _wait_for_port(address: str, timeout: float = 10.0) -> None:
    host, port_text = address.rsplit(":", 1)
    port = int(port_text)
    deadline = time.time() + timeout
    last_error: Exception | None = None
    while time.time() < deadline:
        try:
            with socket.create_connection((host, port), timeout=0.2):
                return
        except OSError as exc:  # pragma: no cover - startup polling
            last_error = exc
            time.sleep(0.1)
    raise RuntimeError(f"cmdagentd did not open {address}: {last_error}")


@pytest.fixture(scope="session")
def repo_root() -> Path:
    return Path(__file__).resolve().parents[2]


@pytest.fixture(scope="session")
def built_cmdagentd(repo_root: Path, tmp_path_factory: pytest.TempPathFactory) -> Path:
    output = tmp_path_factory.mktemp("bin") / ("cmdagentd.exe" if os.name == "nt" else "cmdagentd")
    subprocess.run(
        ["go", "build", "-o", str(output), "./cmd/cmdagentd"],
        cwd=repo_root,
        check=True,
    )
    return output


@pytest.fixture(scope="session")
def dev_certs(repo_root: Path, tmp_path_factory: pytest.TempPathFactory) -> dict[str, Path]:
    cert_dir = tmp_path_factory.mktemp("certs")
    subprocess.run(
        [str(repo_root / "scripts" / "generate-dev-certs.sh"), str(cert_dir)],
        cwd=repo_root,
        check=True,
    )
    return {
        "ca": cert_dir / "ca.crt",
        "server_cert": cert_dir / "server.crt",
        "server_key": cert_dir / "server.key",
        "client_a_cert": cert_dir / "client-a.crt",
        "client_a_key": cert_dir / "client-a.key",
        "client_b_cert": cert_dir / "client-b.crt",
        "client_b_key": cert_dir / "client-b.key",
    }


@pytest.fixture
def managed_daemon(
    built_cmdagentd: Path,
    dev_certs: dict[str, Path],
    tmp_path: Path,
) -> Iterator[dict[str, object]]:
    address = f"127.0.0.1:{_find_free_port()}"
    data_dir = tmp_path / "data"
    audit_log = tmp_path / "audit.log"
    stdout_log = tmp_path / "cmdagentd.stdout.log"
    stderr_log = tmp_path / "cmdagentd.stderr.log"

    with stdout_log.open("wb") as stdout_fh, stderr_log.open("wb") as stderr_fh:
        process = subprocess.Popen(
            [
                str(built_cmdagentd),
                "run",
                "--listen-address",
                address,
                "--server-cert",
                str(dev_certs["server_cert"]),
                "--server-key",
                str(dev_certs["server_key"]),
                "--client-ca",
                str(dev_certs["ca"]),
                "--allowed-client-cn",
                "client-a,client-b",
                "--data-dir",
                str(data_dir),
                "--audit-log",
                str(audit_log),
            ],
            stdout=stdout_fh,
            stderr=stderr_fh,
        )

        client_a = None
        client_b = None

        try:
            _wait_for_port(address)
            client_a = Client(
                address=address,
                ca_cert=str(dev_certs["ca"]),
                client_cert=str(dev_certs["client_a_cert"]),
                client_key=str(dev_certs["client_a_key"]),
            )
            client_b = Client(
                address=address,
                ca_cert=str(dev_certs["ca"]),
                client_cert=str(dev_certs["client_b_cert"]),
                client_key=str(dev_certs["client_b_key"]),
            )
            _wait_for_daemon(client_a)
            yield {
                "address": address,
                "data_dir": data_dir,
                "audit_log": audit_log,
                "stdout_log": stdout_log,
                "stderr_log": stderr_log,
                "client_a": client_a,
                "client_b": client_b,
                "certs": dev_certs,
            }
        finally:
            if client_a is not None:
                with contextlib.suppress(Exception):
                    client_a.close()
            if client_b is not None:
                with contextlib.suppress(Exception):
                    client_b.close()
            if process.poll() is None:
                process.terminate()
                try:
                    process.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=10)


def wait_for_completion(client: Client, execution_id: str, timeout: float = 10.0):
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        last = client.get_execution(execution_id)
        if last.state != agent_pb2.EXECUTION_STATE_RUNNING:
            return last
        time.sleep(0.1)
    raise RuntimeError(f"execution {execution_id} did not finish: {last}")
