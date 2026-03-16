from __future__ import annotations

import os

from cmdra_client import Client


def new_client() -> Client:
    return Client(
        address=os.environ["CMDRA_ADDRESS"],
        ca_cert=os.environ["CMDRA_CA"],
        client_cert=os.environ["CMDRA_CERT"],
        client_key=os.environ["CMDRA_KEY"],
        server_name=os.environ.get("CMDRA_SERVER_NAME") or None,
    )
