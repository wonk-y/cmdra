"""RobotFramework wrapper library around the Python client."""

from __future__ import annotations

from typing import Optional, Sequence

from .client import ClearHistoryResult, Client, DownloadResult, ExecutionDetails

try:
    from robot.api.deco import keyword, library
except ImportError:  # pragma: no cover - optional dependency
    def keyword(func=None, **_kwargs):
        if func is None:
            return lambda inner: inner
        return func

    def library(*_args, **_kwargs):
        def decorator(cls):
            return cls

        return decorator


@library(scope="GLOBAL")
class CmdraLibrary:
    """RobotFramework library exposing the Python SDK methods as keywords."""

    def __init__(
        self,
        address: str,
        ca_cert: str,
        client_cert: str,
        client_key: str,
        server_name: str = "",
        insecure_skip_verify: bool = False,
    ) -> None:
        self._client = Client(
            address=address,
            ca_cert=ca_cert,
            client_cert=client_cert,
            client_key=client_key,
            server_name=server_name or None,
            insecure_skip_verify=insecure_skip_verify,
        )

    @keyword
    def close(self) -> None:
        self._client.close()

    @keyword
    def start_argv(self, binary: str, *args: str):
        return self._client.start_argv(binary, list(args))

    @keyword
    def start_argv_async(self, binary: str, *args: str):
        return self._client.start_argv_async(binary, list(args))

    @keyword
    def start_shell_command(
        self,
        command: str,
        shell_binary: str = "",
        use_pty: bool = False,
        pty_rows: int = 0,
        pty_cols: int = 0,
    ):
        return self._client.start_shell_command(
            command,
            shell_binary=shell_binary,
            use_pty=use_pty,
            pty_rows=pty_rows,
            pty_cols=pty_cols,
        )

    @keyword
    def start_shell_command_async(
        self,
        command: str,
        shell_binary: str = "",
        use_pty: bool = False,
        pty_rows: int = 0,
        pty_cols: int = 0,
    ):
        return self._client.start_shell_command_async(
            command,
            shell_binary=shell_binary,
            use_pty=use_pty,
            pty_rows=pty_rows,
            pty_cols=pty_cols,
        )

    @keyword
    def start_shell_command_with_pty(self, command: str, shell_binary: str = "", pty_rows: int = 0, pty_cols: int = 0):
        return self._client.start_shell_command(command, shell_binary=shell_binary, use_pty=True, pty_rows=pty_rows, pty_cols=pty_cols)

    @keyword
    def start_shell_command_async_with_pty(self, command: str, shell_binary: str = "", pty_rows: int = 0, pty_cols: int = 0):
        return self._client.start_shell_command_async(command, shell_binary=shell_binary, use_pty=True, pty_rows=pty_rows, pty_cols=pty_cols)

    @keyword
    def start_shell_session(self, shell_binary: str, *shell_args: str, use_pty: bool = False, pty_rows: int = 0, pty_cols: int = 0):
        return self._client.start_shell_session(shell_binary, list(shell_args), use_pty=use_pty, pty_rows=pty_rows, pty_cols=pty_cols)

    @keyword
    def start_shell_session_async(self, shell_binary: str, *shell_args: str, use_pty: bool = False, pty_rows: int = 0, pty_cols: int = 0):
        return self._client.start_shell_session_async(shell_binary, list(shell_args), use_pty=use_pty, pty_rows=pty_rows, pty_cols=pty_cols)

    @keyword
    def start_shell_session_with_pty(self, shell_binary: str, *shell_args: str, pty_rows: int = 0, pty_cols: int = 0):
        return self._client.start_shell_session(shell_binary, list(shell_args), use_pty=True, pty_rows=pty_rows, pty_cols=pty_cols)

    @keyword
    def start_shell_session_async_with_pty(self, shell_binary: str, *shell_args: str, pty_rows: int = 0, pty_cols: int = 0):
        return self._client.start_shell_session_async(shell_binary, list(shell_args), use_pty=True, pty_rows=pty_rows, pty_cols=pty_cols)

    @keyword
    def get_execution(self, execution_id: str):
        return self._client.get_execution(execution_id)

    @keyword
    def list_executions(self, running_only: Optional[bool] = None):
        return self._client.list_executions(running_only)

    @keyword
    def delete_execution(self, execution_id: str) -> str:
        return self._client.delete_execution(execution_id)

    @keyword
    def clear_history(self) -> ClearHistoryResult:
        return self._client.clear_history()

    @keyword
    def get_execution_with_output(self, execution_id: str, discard_after_read: bool = False) -> ExecutionDetails:
        return self._client.get_execution_with_output(execution_id, discard_after_read)

    @keyword
    def cancel_execution(self, execution_id: str, grace_period_seconds: int = 5):
        return self._client.cancel_execution(execution_id, grace_period_seconds)

    @keyword
    def read_output(self, execution_id: str, offset: int = 0, discard_after_read: bool = False):
        return self._client.read_output(execution_id, offset, discard_after_read)

    @keyword
    def write_stdin(self, execution_id: str, data: str = "", eof: bool = False) -> None:
        self._client.write_stdin(execution_id, data.encode(), eof)

    @keyword
    def upload_file(self, local_path: str, remote_path: str, overwrite: bool = True, file_mode: Optional[int] = None):
        return self._client.upload_file(local_path, remote_path, overwrite, file_mode)

    @keyword
    def upload_file_async(self, local_path: str, remote_path: str, overwrite: bool = True, file_mode: Optional[int] = None):
        return self._client.upload_file_async(local_path, remote_path, overwrite, file_mode)

    @keyword
    def download_file(self, remote_path: str, local_path: str, chunk_size: int = 32 * 1024) -> DownloadResult:
        return self._client.download_file(remote_path, local_path, chunk_size)

    @keyword
    def download_file_async(self, remote_path: str, local_path: str, chunk_size: int = 32 * 1024):
        return self._client.download_file_async(remote_path, local_path, chunk_size)

    @keyword
    def download_archive(self, paths: Sequence[str], local_path: str, chunk_size: int = 32 * 1024) -> DownloadResult:
        return self._client.download_archive(paths, local_path, chunk_size)

    @keyword
    def download_archive_async(self, paths: Sequence[str], local_path: str, chunk_size: int = 32 * 1024):
        return self._client.download_archive_async(paths, local_path, chunk_size)
