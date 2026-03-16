# Development Assets

Local development materials live under `dev/`.

- `dev/certs/`: development CA, server certificate, and client certificates
- `dev/data/`: runtime SQLite data and audit output
- `dev/cmdrad.json`: sample local daemon configuration

The repository documentation uses these paths consistently for local examples so source files and generated runtime state stay separated.

`dev/data/` is where history cleanup operations act. The `delete` and `clear-history` commands remove persisted SQLite history from the remote daemon's configured data directory.
