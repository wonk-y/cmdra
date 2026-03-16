from __future__ import annotations

import argparse

from _client import new_client


parser = argparse.ArgumentParser()
parser.add_argument("--remote", required=True)
parser.add_argument("--local", required=True)
args = parser.parse_args()

client = new_client()
result = client.download_file(args.remote, args.local)
print(f"transfer_id={result.transfer_id} bytes={result.bytes_written}")
client.close()
