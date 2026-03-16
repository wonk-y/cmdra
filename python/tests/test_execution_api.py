from __future__ import annotations

from cmdagent_client.gen.agent.v1 import agent_pb2

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

    assert any(b"attached-async-output" in chunk for chunk in output_chunks)
