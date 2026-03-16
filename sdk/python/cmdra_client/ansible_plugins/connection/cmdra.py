"""Ansible connection plugin backed by cmdrad."""

from __future__ import annotations

import os
import shlex
import time

from ansible.errors import AnsibleConnectionFailure, AnsibleFileNotFound
from ansible.plugins.connection import ConnectionBase

from cmdra_client.client import Client
from cmdra_client.gen.agent.v1 import agent_pb2

DOCUMENTATION = r"""
connection: cmdra
short_description: Run Ansible modules through cmdrad
description:
  - Uses the cmdra Python SDK to execute commands and transfer files over mTLS.
options:
  address:
    description: cmdrad host:port
    vars:
      - name: ansible_cmdra_address
  ca_cert:
    description: CA certificate path
    vars:
      - name: ansible_cmdra_ca_cert
  client_cert:
    description: Client certificate path
    vars:
      - name: ansible_cmdra_client_cert
  client_key:
    description: Client private key path
    vars:
      - name: ansible_cmdra_client_key
  server_name:
    description: TLS server name override
    vars:
      - name: ansible_cmdra_server_name
"""


class Connection(ConnectionBase):
    transport = "cmdra"
    has_pipelining = True

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._client = None

    def _connect(self):
        if self._client is None:
            try:
                self._client = Client(
                    address=self.get_option("address"),
                    ca_cert=self.get_option("ca_cert"),
                    client_cert=self.get_option("client_cert"),
                    client_key=self.get_option("client_key"),
                    server_name=self.get_option("server_name") or None,
                )
            except Exception as exc:  # pragma: no cover - ansible integration path
                raise AnsibleConnectionFailure(str(exc)) from exc
        return self

    def close(self):
        if self._client is not None:
            self._client.close()
            self._client = None

    def exec_command(self, cmd, in_data=None, sudoable=True):
        self._connect()
        try:
            execution = self._client.start_shell_command(cmd)
            metadata = self._wait_for_completion(execution.execution_id)
            output = self._client.read_output(execution.execution_id)
        except Exception as exc:  # pragma: no cover - ansible integration path
            raise AnsibleConnectionFailure(str(exc)) from exc

        stdout_parts = []
        stderr_parts = []
        for chunk in output:
            if chunk.eof:
                continue
            if chunk.source == agent_pb2.OUTPUT_SOURCE_STDERR:
                stderr_parts.append(chunk.data)
            else:
                stdout_parts.append(chunk.data)
        return metadata.exit_code, b"".join(stdout_parts), b"".join(stderr_parts)

    def put_file(self, in_path, out_path):
        self._connect()
        if not os.path.exists(in_path):
            raise AnsibleFileNotFound(f"local path does not exist: {in_path}")
        try:
            self._client.upload_file(in_path, out_path)
        except Exception as exc:  # pragma: no cover - ansible integration path
            raise AnsibleConnectionFailure(str(exc)) from exc

    def fetch_file(self, in_path, out_path):
        self._connect()
        try:
            self._client.download_file(in_path, out_path)
        except Exception as exc:  # pragma: no cover - ansible integration path
            raise AnsibleConnectionFailure(str(exc)) from exc

    def put_data(self, out_path, data, mode=None, **kwargs):
        self._connect()
        import tempfile

        with tempfile.NamedTemporaryFile(delete=False) as fh:
            fh.write(data)
            temp_path = fh.name
        try:
            self._client.upload_file(temp_path, out_path, file_mode=mode)
        finally:
            try:
                os.unlink(temp_path)
            except OSError:
                pass

    def _shell(self):
        return shlex

    def _wait_for_completion(self, execution_id, timeout=30.0):
        deadline = time.time() + timeout
        last = None
        while time.time() < deadline:
            last = self._client.get_execution(execution_id)
            if last.state != agent_pb2.EXECUTION_STATE_RUNNING:
                return last
            time.sleep(0.1)
        raise AnsibleConnectionFailure(f"execution {execution_id} did not finish: {last}")
