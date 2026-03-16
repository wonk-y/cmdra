from __future__ import annotations

import grpc
import pytest

from .conftest import wait_for_completion


def _assert_rpc_code(exc: pytest.ExceptionInfo[grpc.RpcError], expected: grpc.StatusCode) -> None:
    assert exc.value.code() == expected, f"expected {expected}, got {exc.value.code()} ({exc.value.details()})"


def test_cross_identity_authorization(managed_daemon):
    client_a = managed_daemon["client_a"]
    client_b = managed_daemon["client_b"]

    execution = client_a.start_shell_command("printf 'owned-by-a\\n'")
    wait_for_completion(client_a, execution.execution_id)

    with pytest.raises(grpc.RpcError) as exc:
        client_b.get_execution(execution.execution_id)
    _assert_rpc_code(exc, grpc.StatusCode.PERMISSION_DENIED)

    with pytest.raises(grpc.RpcError) as exc:
        client_b.read_output(execution.execution_id)
    _assert_rpc_code(exc, grpc.StatusCode.PERMISSION_DENIED)

    with pytest.raises(grpc.RpcError) as exc:
        client_b.delete_execution(execution.execution_id)
    _assert_rpc_code(exc, grpc.StatusCode.PERMISSION_DENIED)
