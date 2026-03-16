from __future__ import annotations

import os

from cmdagent_client import Client


def new_client() -> Client:
    return Client(
        address=os.environ["CMDAGENT_ADDRESS"],
        ca_cert=os.environ["CMDAGENT_CA"],
        client_cert=os.environ["CMDAGENT_CERT"],
        client_key=os.environ["CMDAGENT_KEY"],
        server_name=os.environ.get("CMDAGENT_SERVER_NAME") or None,
    )
