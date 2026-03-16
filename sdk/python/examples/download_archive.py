from __future__ import annotations

import argparse

from _client import new_client


parser = argparse.ArgumentParser()
parser.add_argument("--path", action="append", required=True)
parser.add_argument("--local", required=True)
args = parser.parse_args()

client = new_client()
result = client.download_archive(args.path, args.local)
print(f"transfer_id={result.transfer_id} bytes={result.bytes_written}")
client.close()
