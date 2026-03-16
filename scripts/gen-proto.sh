#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

export PATH="$HOME/go/bin:$PATH"

for tool in protoc protoc-gen-go protoc-gen-go-grpc; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "missing required tool: $tool" >&2
    exit 1
  fi
done

if [ ! -x .venv/bin/python ]; then
  echo "missing .venv/bin/python" >&2
  exit 1
fi

if ! .venv/bin/python -c 'import grpc_tools.protoc' >/dev/null 2>&1; then
  echo "missing grpcio-tools in .venv; run: .venv/bin/pip install grpcio-tools" >&2
  exit 1
fi

protoc -I proto \
  --go_out=. --go_opt=module=cmdra \
  --go-grpc_out=. --go-grpc_opt=module=cmdra \
  proto/agent/v1/agent.proto

rm -f sdk/python/cmdra_client/gen/agent/v1/agent_pb2.py sdk/python/cmdra_client/gen/agent/v1/agent_pb2_grpc.py
.venv/bin/python -m grpc_tools.protoc \
  -I proto \
  --python_out=sdk/python/cmdra_client/gen \
  --grpc_python_out=sdk/python/cmdra_client/gen \
  proto/agent/v1/agent.proto

python_file="sdk/python/cmdra_client/gen/agent/v1/agent_pb2_grpc.py"
python_pb2_file="sdk/python/cmdra_client/gen/agent/v1/agent_pb2.py"
python3 - <<'PY'
from pathlib import Path
for path in [Path("sdk/python/cmdra_client/gen/agent/v1/agent_pb2_grpc.py")]:
    data = path.read_text(encoding="utf-8")
    data = data.replace("from agent.v1 import agent_pb2 as agent_dot_v1_dot_agent__pb2", "from . import agent_pb2 as agent_dot_v1_dot_agent__pb2")
    path.write_text(data, encoding="utf-8")
PY

gofmt -w gen/agent/v1/*.go
.venv/bin/python -m py_compile "$python_pb2_file" "$python_file"

echo "protobuf stubs regenerated"
