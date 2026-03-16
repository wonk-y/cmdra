from __future__ import annotations

from _client import new_client
from cmdagent_client import format_execution


client = new_client()
for execution in client.list_executions():
    print(format_execution(execution))
    print()
client.close()
