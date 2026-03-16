from __future__ import annotations

import argparse

from _client import new_client
from cmdra_client import format_execution


parser = argparse.ArgumentParser()
parser.add_argument("--id", required=True)
args = parser.parse_args()

client = new_client()
details = client.get_execution_with_output(args.id)
print(format_execution(details.execution))
print()
for chunk in details.output:
    if chunk.eof:
        continue
    print(chunk.data.decode("utf-8", errors="replace"), end="")
client.close()
