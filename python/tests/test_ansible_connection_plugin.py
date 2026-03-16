from __future__ import annotations

import os
import subprocess
from pathlib import Path

import pytest

from cmdagent_client import get_ansible_plugin_path


@pytest.mark.skipif(not Path(".venv/bin/ansible-playbook").exists(), reason="ansible-playbook is not installed")
def test_ansible_connection_plugin(managed_daemon, tmp_path: Path, repo_root: Path):
    inventory = tmp_path / "inventory.ini"
    playbook = tmp_path / "playbook.yml"
    fetch_dir = tmp_path / "fetched"
    fetch_dir.mkdir()

    certs = managed_daemon["certs"]
    inventory.write_text(
        "\n".join(
            [
                "[cmdagent]",
                "target ansible_connection=cmdagent",
                f"target ansible_cmdagent_address={managed_daemon['address']}",
                f"target ansible_cmdagent_ca_cert={certs['ca']}",
                f"target ansible_cmdagent_client_cert={certs['client_a_cert']}",
                f"target ansible_cmdagent_client_key={certs['client_a_key']}",
                "",
            ]
        ),
        encoding="utf-8",
    )

    playbook.write_text(
        "\n".join(
            [
                "- hosts: cmdagent",
                "  gather_facts: false",
                "  tasks:",
                "    - name: run command",
                "      command: /bin/echo ansible-ok",
                "    - name: copy file",
                "      copy:",
                "        content: copied-by-ansible",
                "        dest: " + str(tmp_path / "remote-file.txt"),
                "    - name: fetch file",
                "      fetch:",
                "        src: " + str(tmp_path / "remote-file.txt"),
                "        dest: " + str(fetch_dir) + "/",
                "        flat: true",
                "",
            ]
        ),
        encoding="utf-8",
    )

    env = os.environ.copy()
    env["PYTHONPATH"] = str(repo_root / "python")
    env["ANSIBLE_CONNECTION_PLUGINS"] = get_ansible_plugin_path()
    env["ANSIBLE_LOCAL_TEMP"] = str(tmp_path / "ansible-local")
    env["ANSIBLE_REMOTE_TEMP"] = str(tmp_path / "ansible-remote")
    Path(env["ANSIBLE_LOCAL_TEMP"]).mkdir(parents=True, exist_ok=True)
    Path(env["ANSIBLE_REMOTE_TEMP"]).mkdir(parents=True, exist_ok=True)

    subprocess.run(
        [str(repo_root / ".venv" / "bin" / "ansible-playbook"), "-i", str(inventory), str(playbook)],
        check=True,
        cwd=repo_root,
        env=env,
    )

    fetched = fetch_dir / "remote-file.txt"
    assert fetched.read_text(encoding="utf-8") == "copied-by-ansible"
