from __future__ import annotations

import time

import grpc

from cmdra_client.gen.agent.v1 import agent_pb2

from .conftest import wait_for_completion


def test_start_argv_and_list_metadata(managed_daemon):
    client = managed_daemon["client_a"]
    execution = client.start_argv("/bin/echo", ["hello-world"])
    finished = wait_for_completion(client, execution.execution_id)

    assert finished.state == agent_pb2.EXECUTION_STATE_EXITED
    assert finished.command_argv[0] == "/bin/echo"
    assert finished.command_argv[1:] == ["hello-world"]
    assert finished.exit_code == 0

    executions = client.list_executions()
    by_id = {item.execution_id: item for item in executions}
    assert execution.execution_id in by_id
    assert list(by_id[execution.execution_id].command_argv) == ["/bin/echo", "hello-world"]


def test_async_start_argv_and_shell_command(managed_daemon):
    client = managed_daemon["client_a"]

    argv_execution = client.start_argv_async("/bin/echo", ["hello-async-argv"]).result(timeout=10)
    argv_finished = wait_for_completion(client, argv_execution.execution_id)
    argv_details = client.get_execution_with_output(argv_execution.execution_id)

    assert argv_finished.state == agent_pb2.EXECUTION_STATE_EXITED
    assert b"hello-async-argv\n" == b"".join(chunk.data for chunk in argv_details.output if not chunk.eof)

    shell_execution = client.start_shell_command_async("printf 'hello-async-shell\\n'").result(timeout=10)
    shell_finished = wait_for_completion(client, shell_execution.execution_id)
    shell_details = client.get_execution_with_output(shell_execution.execution_id)

    assert shell_finished.state == agent_pb2.EXECUTION_STATE_EXITED
    assert b"hello-async-shell\n" == b"".join(chunk.data for chunk in shell_details.output if not chunk.eof)


def test_get_execution_with_output_for_shell_command(managed_daemon):
    client = managed_daemon["client_a"]
    execution = client.start_shell_command("printf 'stdout-line\\n'; printf 'stderr-line\\n' >&2")
    wait_for_completion(client, execution.execution_id)

    details = client.get_execution_with_output(execution.execution_id)
    combined = b"".join(chunk.data for chunk in details.output if not chunk.eof)

    assert details.execution.command_shell == "printf 'stdout-line\\n'; printf 'stderr-line\\n' >&2"
    assert b"stdout-line" in combined
    assert b"stderr-line" in combined


def test_attach_shell_session(managed_daemon):
    client = managed_daemon["client_a"]
    execution = client.start_shell_session("/bin/sh")
    session = client.attach(execution.execution_id, replay_buffered=True)

    ack = session.recv()
    assert ack.HasField("ack")
    session.send_stdin(b"printf 'attached-output\\n'\nexit\n", eof=True)

    output_chunks = []
    while True:
        event = session.recv()
        if event.HasField("output"):
            output_chunks.append(event.output.data)
        if event.HasField("exit"):
            break

    assert any(b"attached-output" in chunk for chunk in output_chunks)


def test_write_stdin_shell_command(managed_daemon):
    client = managed_daemon["client_a"]
    execution = client.start_shell_command("read first; read second; printf '%s-%s\\n' \"$first\" \"$second\"", shell_binary="/bin/sh")

    client.write_stdin(execution.execution_id, b"alpha\n", eof=False)
    client.write_stdin(execution.execution_id, b"beta\n", eof=True)

    wait_for_completion(client, execution.execution_id)
    details = client.get_execution_with_output(execution.execution_id)
    combined = b"".join(chunk.data for chunk in details.output if not chunk.eof)

    assert b"alpha-beta" in combined


def test_write_stdin_shell_session(managed_daemon):
    client = managed_daemon["client_a"]
    execution = client.start_shell_session("/bin/sh")

    client.write_stdin(execution.execution_id, b"printf 'stdin-session\\n'\n", eof=False)
    client.write_stdin(execution.execution_id, b"exit\n", eof=True)

    wait_for_completion(client, execution.execution_id)
    details = client.get_execution_with_output(execution.execution_id)
    combined = b"".join(chunk.data for chunk in details.output if not chunk.eof)

    assert b"stdin-session" in combined


def test_shell_command_with_pty(managed_daemon):
    client = managed_daemon["client_a"]
    execution = client.start_shell_command("printf 'python-pty\\n'", shell_binary="/bin/sh", use_pty=True)
    finished = wait_for_completion(client, execution.execution_id)
    details = client.get_execution_with_output(execution.execution_id)

    assert finished.uses_pty is True
    assert b"python-pty" in b"".join(chunk.data for chunk in details.output if not chunk.eof)


def test_shell_session_with_pty_resize(managed_daemon):
    client = managed_daemon["client_a"]
    execution = client.start_shell_session("/bin/sh", use_pty=True, pty_rows=24, pty_cols=80)

    assert execution.uses_pty is True
    assert execution.pty_rows == 24
    assert execution.pty_cols == 80

    session = client.attach(execution.execution_id, replay_buffered=True)
    ack = session.recv()
    assert ack.HasField("ack")
    assert ack.ack.execution.pty_rows == 24
    assert ack.ack.execution.pty_cols == 80

    session.resize_pty(40, 100)

    deadline = time.time() + 5
    while time.time() < deadline:
        current = client.get_execution(execution.execution_id)
        if current.pty_rows == 40 and current.pty_cols == 100:
            break
        time.sleep(0.1)
    else:  # pragma: no cover - defensive
        current = client.get_execution(execution.execution_id)
        raise AssertionError(f"expected PTY size 40x100, got {current.pty_rows}x{current.pty_cols}")

    session.send_stdin(b"exit\n", eof=True)
    wait_for_completion(client, execution.execution_id)


def test_async_start_shell_session_and_attach(managed_daemon):
    client = managed_daemon["client_a"]
    execution = client.start_shell_session_async("/bin/sh").result(timeout=10)
    session = client.attach(execution.execution_id, replay_buffered=True)

    ack = session.recv()
    assert ack.HasField("ack")
    session.send_stdin(b"printf 'attached-async-output\\n'\nexit\n", eof=True)

    output_chunks = []
    while True:
        event = session.recv()
        if event.HasField("output"):
            output_chunks.append(event.output.data)
        if event.HasField("exit"):
            break

    if not any(b"attached-async-output" in chunk for chunk in output_chunks):
        details = client.get_execution_with_output(execution.execution_id)
        output_chunks.extend(chunk.data for chunk in details.output if not chunk.eof)
    assert any(b"attached-async-output" in chunk for chunk in output_chunks)


def test_delete_execution_and_clear_history(managed_daemon, tmp_path):
    client = managed_daemon["client_a"]

    deleted_execution = client.start_argv("/bin/echo", ["delete-python"])
    wait_for_completion(client, deleted_execution.execution_id)
    assert client.delete_execution(deleted_execution.execution_id) == deleted_execution.execution_id

    try:
        client.get_execution(deleted_execution.execution_id)
    except grpc.RpcError as exc:
        assert exc.code() == grpc.StatusCode.NOT_FOUND
    else:  # pragma: no cover - defensive
        raise AssertionError("deleted execution should not be returned")

    finished_execution = client.start_argv("/bin/echo", ["clear-python"])
    wait_for_completion(client, finished_execution.execution_id)

    upload_source = tmp_path / "clear-upload.txt"
    upload_source.write_text("clear-history python\n", encoding="utf-8")
    remote_path = tmp_path / "clear-remote.txt"
    upload = client.upload_file(str(upload_source), str(remote_path))

    running_execution = client.start_shell_command("sleep 30", shell_binary="/bin/sh")

    result = client.clear_history()
    assert result.deleted_count == 2
    assert result.skipped_running_count == 1

    for deleted_id in (finished_execution.execution_id, upload.transfer_id):
        try:
            client.get_execution(deleted_id)
        except grpc.RpcError as exc:
            assert exc.code() == grpc.StatusCode.NOT_FOUND
        else:  # pragma: no cover - defensive
            raise AssertionError(f"{deleted_id} should be gone after clear_history")

    running_meta = client.get_execution(running_execution.execution_id)
    assert running_meta.state == agent_pb2.EXECUTION_STATE_RUNNING

    try:
        client.delete_execution(running_execution.execution_id)
    except grpc.RpcError as exc:
        assert exc.code() == grpc.StatusCode.FAILED_PRECONDITION
    else:  # pragma: no cover - defensive
        raise AssertionError("deleting a running execution should fail")

    client.cancel_execution(running_execution.execution_id)
