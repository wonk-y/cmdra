from __future__ import annotations

from _client import new_client


client = new_client()
execution = client.start_argv("/bin/echo", ["hello-from-python-sdk"])
print(execution.execution_id)
client.close()
