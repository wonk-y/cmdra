"""Python SDK for cmdagentd."""

from __future__ import annotations

import concurrent.futures
import queue
from pathlib import Path
from typing import Iterable, Iterator, Optional, Sequence

import grpc

from .gen.agent.v1 import agent_pb2, agent_pb2_grpc


class DownloadResult:
    """Final metadata from a streaming download operation."""

    def __init__(self, bytes_written: int, transfer_id: str) -> None:
        self.bytes_written = bytes_written
        self.transfer_id = transfer_id


class ExecutionDetails:
    """Execution metadata plus replayed output chunks."""

    def __init__(self, execution: agent_pb2.Execution, output: list[agent_pb2.OutputChunk]) -> None:
        self.execution = execution
        self.output = output


class AttachSession:
    """Bidirectional attach session wrapper."""

    def __init__(self, call: Iterator[agent_pb2.AttachEvent], requests: "queue.Queue[Optional[agent_pb2.AttachRequest]]") -> None:
        self._call = call
        self._requests = requests

    def send_stdin(self, data: bytes, eof: bool = False) -> None:
        self._requests.put(
            agent_pb2.AttachRequest(
                stdin=agent_pb2.StdinChunk(data=data, eof=eof),
            )
        )

    def cancel_execution(self) -> None:
        self._requests.put(
            agent_pb2.AttachRequest(
                control=agent_pb2.AttachControl(cancel_execution=True),
            )
        )

    def recv(self) -> agent_pb2.AttachEvent:
        return next(self._call)

    def __iter__(self) -> Iterator[agent_pb2.AttachEvent]:
        return iter(self._call)

    def close(self) -> None:
        self._requests.put(None)


class Client:
    """High-level Python client for cmdagentd."""

    def __init__(
        self,
        address: str,
        ca_cert: str,
        client_cert: str,
        client_key: str,
        server_name: Optional[str] = None,
        insecure_skip_verify: bool = False,
        channel_options: Optional[Sequence[tuple[str, object]]] = None,
        executor: Optional[concurrent.futures.Executor] = None,
    ) -> None:
        self.address = address
        self._owns_executor = executor is None
        self._executor = executor or concurrent.futures.ThreadPoolExecutor(max_workers=4)
        with open(ca_cert, "rb") as fh:
            root_certificates = fh.read()
        with open(client_cert, "rb") as fh:
            certificate_chain = fh.read()
        with open(client_key, "rb") as fh:
            private_key = fh.read()

        credentials = grpc.ssl_channel_credentials(
            root_certificates=root_certificates,
            private_key=private_key,
            certificate_chain=certificate_chain,
        )
        options = list(channel_options or [])
        if server_name:
            options.append(("grpc.ssl_target_name_override", server_name))
        if insecure_skip_verify:
            options.append(("grpc.ssl_target_name_override", server_name or "cmdagentd"))
        self._channel = grpc.secure_channel(address, credentials, options=options)
        self._stub = agent_pb2_grpc.AgentServiceStub(self._channel)

    def close(self) -> None:
        self._channel.close()
        if self._owns_executor:
            self._executor.shutdown(wait=False, cancel_futures=False)

    def start_argv(self, binary: str, args: Optional[Sequence[str]] = None) -> agent_pb2.Execution:
        response = self._stub.StartCommand(
            agent_pb2.StartCommandRequest(
                argv=agent_pb2.ArgvCommand(binary=binary, args=list(args or []))
            )
        )
        return response.execution

    def start_argv_async(
        self,
        binary: str,
        args: Optional[Sequence[str]] = None,
    ) -> concurrent.futures.Future[agent_pb2.Execution]:
        return self._executor.submit(self.start_argv, binary, list(args or []))

    def start_shell_command(self, command: str, shell_binary: str = "") -> agent_pb2.Execution:
        response = self._stub.StartCommand(
            agent_pb2.StartCommandRequest(
                shell=agent_pb2.ShellCommand(shell_binary=shell_binary, command=command)
            )
        )
        return response.execution

    def start_shell_command_async(
        self,
        command: str,
        shell_binary: str = "",
    ) -> concurrent.futures.Future[agent_pb2.Execution]:
        return self._executor.submit(self.start_shell_command, command, shell_binary)

    def start_shell_session(self, shell_binary: str, shell_args: Optional[Sequence[str]] = None) -> agent_pb2.Execution:
        response = self._stub.StartShell(
            agent_pb2.StartShellRequest(shell_binary=shell_binary, shell_args=list(shell_args or []))
        )
        return response.execution

    def start_shell_session_async(
        self,
        shell_binary: str,
        shell_args: Optional[Sequence[str]] = None,
    ) -> concurrent.futures.Future[agent_pb2.Execution]:
        return self._executor.submit(self.start_shell_session, shell_binary, list(shell_args or []))

    def get_execution(self, execution_id: str) -> agent_pb2.Execution:
        response = self._stub.GetExecution(agent_pb2.GetExecutionRequest(execution_id=execution_id))
        return response.execution

    def list_executions(self, running_only: Optional[bool] = None) -> list[agent_pb2.Execution]:
        request = agent_pb2.ListExecutionsRequest()
        if running_only is not None:
            request.running_only = running_only
        response = self._stub.ListExecutions(request)
        return list(response.executions)

    def cancel_execution(self, execution_id: str, grace_period_seconds: int = 5) -> agent_pb2.Execution:
        response = self._stub.CancelExecution(
            agent_pb2.CancelExecutionRequest(
                execution_id=execution_id,
                grace_period_seconds=grace_period_seconds,
            )
        )
        return response.execution

    def read_output(self, execution_id: str, offset: int = 0, discard_after_read: bool = False) -> list[agent_pb2.OutputChunk]:
        return list(
            self._stub.ReadOutput(
                agent_pb2.ReadOutputRequest(
                    execution_id=execution_id,
                    offset=offset,
                    discard_after_read=discard_after_read,
                )
            )
        )

    def get_execution_with_output(self, execution_id: str, discard_after_read: bool = False) -> ExecutionDetails:
        execution = self.get_execution(execution_id)
        output = self.read_output(execution_id, discard_after_read=discard_after_read)
        return ExecutionDetails(execution, output)

    def attach(self, execution_id: str, replay_buffered: bool = True, replay_from_offset: int = 0) -> AttachSession:
        requests: "queue.Queue[Optional[agent_pb2.AttachRequest]]" = queue.Queue()

        def iterator() -> Iterator[agent_pb2.AttachRequest]:
            while True:
                item = requests.get()
                if item is None:
                    return
                yield item

        call = self._stub.Attach(iterator())
        requests.put(
            agent_pb2.AttachRequest(
                start=agent_pb2.AttachStart(
                    execution_id=execution_id,
                    replay_buffered=replay_buffered,
                    replay_from_offset=replay_from_offset,
                )
            )
        )
        return AttachSession(call, requests)

    def upload_file(
        self,
        local_path: str,
        remote_path: str,
        overwrite: bool = True,
        file_mode: Optional[int] = None,
    ) -> agent_pb2.UploadFileResponse:
        local = Path(local_path)
        if file_mode is None:
            file_mode = local.stat().st_mode & 0o777

        def iterator() -> Iterator[agent_pb2.UploadFileRequest]:
            yield agent_pb2.UploadFileRequest(
                metadata=agent_pb2.UploadFileMetadata(
                    path=remote_path,
                    overwrite=overwrite,
                    file_mode=file_mode,
                    client_local_path=str(local),
                )
            )
            with local.open("rb") as fh:
                while True:
                    chunk = fh.read(32 * 1024)
                    if not chunk:
                        break
                    yield agent_pb2.UploadFileRequest(chunk=chunk)

        return self._stub.UploadFile(iterator())

    def upload_file_async(
        self,
        local_path: str,
        remote_path: str,
        overwrite: bool = True,
        file_mode: Optional[int] = None,
    ) -> concurrent.futures.Future[agent_pb2.UploadFileResponse]:
        return self._executor.submit(self.upload_file, local_path, remote_path, overwrite, file_mode)

    def download_file(self, remote_path: str, local_path: str, chunk_size: int = 32 * 1024) -> DownloadResult:
        stream = self._stub.DownloadFile(
            agent_pb2.DownloadFileRequest(
                path=remote_path,
                chunk_size=chunk_size,
                client_local_path=local_path,
            )
        )
        return self._consume_file_stream(local_path, stream)

    def download_file_async(self, remote_path: str, local_path: str, chunk_size: int = 32 * 1024) -> concurrent.futures.Future[DownloadResult]:
        return self._executor.submit(self.download_file, remote_path, local_path, chunk_size)

    def download_archive(
        self,
        paths: Sequence[str],
        local_path: str,
        chunk_size: int = 32 * 1024,
    ) -> DownloadResult:
        stream = self._stub.DownloadArchive(
            agent_pb2.DownloadArchiveRequest(
                paths=list(paths),
                format=agent_pb2.ARCHIVE_FORMAT_ZIP,
                chunk_size=chunk_size,
                client_local_path=local_path,
            )
        )
        return self._consume_file_stream(local_path, stream)

    def download_archive_async(
        self,
        paths: Sequence[str],
        local_path: str,
        chunk_size: int = 32 * 1024,
    ) -> concurrent.futures.Future[DownloadResult]:
        return self._executor.submit(self.download_archive, list(paths), local_path, chunk_size)

    def _consume_file_stream(self, local_path: str, stream: Iterable[agent_pb2.FileChunk]) -> DownloadResult:
        output = Path(local_path)
        output.parent.mkdir(parents=True, exist_ok=True)
        transfer_id = ""
        bytes_written = 0
        with output.open("wb") as fh:
            for chunk in stream:
                if chunk.transfer_id:
                    transfer_id = chunk.transfer_id
                if chunk.data:
                    fh.write(chunk.data)
                    bytes_written += len(chunk.data)
                if chunk.eof:
                    break
        return DownloadResult(bytes_written=bytes_written, transfer_id=transfer_id)


def format_execution(execution: agent_pb2.Execution) -> str:
    """Render execution metadata in a compact human-readable format."""

    lines = [
        f"ID: {execution.execution_id}",
        f"Kind: {execution.kind}",
        f"State: {execution.state}",
        f"Owner CN: {execution.owner_cn}",
    ]
    if execution.pid:
        lines.append(f"PID: {execution.pid}")
    if execution.command_argv:
        lines.append(f"Command Argv: {' '.join(execution.command_argv)}")
    if execution.command_shell:
        lines.append(f"Command Shell: {execution.command_shell}")
    if execution.last_upload_local_path:
        lines.append(f"Upload Local Path: {execution.last_upload_local_path}")
    if execution.last_upload_remote_path:
        lines.append(f"Upload Remote Path: {execution.last_upload_remote_path}")
    if execution.last_upload_transfer_id:
        lines.append(f"Upload Transfer ID: {execution.last_upload_transfer_id}")
    if execution.last_download_local_path:
        lines.append(f"Download Local Path: {execution.last_download_local_path}")
    if execution.last_download_remote_path:
        lines.append(f"Download Remote Path: {execution.last_download_remote_path}")
    if execution.last_download_transfer_id:
        lines.append(f"Download Transfer ID: {execution.last_download_transfer_id}")
    if execution.transfer_direction:
        lines.append(f"Transfer Direction: {execution.transfer_direction}")
        lines.append(f"Transfer Progress Bytes: {execution.transfer_progress_bytes}")
        if execution.transfer_total_bytes:
            lines.append(f"Transfer Total Bytes: {execution.transfer_total_bytes}")
    lines.append(f"Output Size Bytes: {execution.output_size_bytes}")
    lines.append(f"Exit Code: {execution.exit_code}")
    if execution.signal:
        lines.append(f"Signal: {execution.signal}")
    if execution.error_message:
        lines.append(f"Error: {execution.error_message}")
    return "\n".join(lines)
