from __future__ import annotations

import os
import subprocess
from pathlib import Path


def test_robot_library_smoke(managed_daemon, repo_root: Path, tmp_path: Path):
    robot_bin = repo_root / ".venv" / "bin" / "robot"
    if not robot_bin.exists():
        raise RuntimeError("robotframework is not installed in .venv")

    certs = managed_daemon["certs"]
    env = os.environ.copy()
    env["PYTHONPATH"] = str(repo_root / "sdk" / "python")

    output_dir = tmp_path / "robot-output"
    output_dir.mkdir()

    subprocess.run(
        [
            str(robot_bin),
            "--outputdir",
            str(output_dir),
            "--variable",
            f"ADDRESS:{managed_daemon['address']}",
            "--variable",
            f"CA_CERT:{certs['ca']}",
            "--variable",
            f"CLIENT_CERT:{certs['client_a_cert']}",
            "--variable",
            f"CLIENT_KEY:{certs['client_a_key']}",
            "sdk/python/examples/robot_smoke.robot",
            "sdk/python/examples/robot_ci.robot",
        ],
        check=True,
        cwd=repo_root,
        env=env,
    )
