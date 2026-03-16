"""Helpers for locating the packaged Ansible plugin."""

from __future__ import annotations

from pathlib import Path


def get_ansible_plugin_path() -> str:
    """Return the packaged Ansible plugin root directory."""

    return str(Path(__file__).resolve().parent / "ansible_plugins")
