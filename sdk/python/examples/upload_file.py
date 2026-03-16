from __future__ import annotations

import argparse

from _client import new_client


parser = argparse.ArgumentParser()
parser.add_argument("--local", required=True)
parser.add_argument("--remote", required=True)
args = parser.parse_args()

client = new_client()
response = client.upload_file(args.local, args.remote)
print(f"transfer_id={response.transfer_id} bytes={response.bytes_written} sha256={response.sha256}")
client.close()
