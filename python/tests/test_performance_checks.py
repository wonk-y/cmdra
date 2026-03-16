from __future__ import annotations

import os
import time

import pytest

from .conftest import wait_for_completion


@pytest.mark.skipif(os.environ.get("CMDAGENT_RUN_PERF") != "1", reason="set CMDAGENT_RUN_PERF=1 to run timing checks")
def test_start_command_latency(managed_daemon):
    client = managed_daemon["client_a"]

    start = time.perf_counter()
    execution = client.start_argv("/bin/echo", ["perf-check"])
    wait_for_completion(client, execution.execution_id)
    elapsed = time.perf_counter() - start

    assert elapsed < 5.0
