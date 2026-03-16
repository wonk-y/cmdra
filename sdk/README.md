# SDK Layout

Language-specific SDK assets live under `sdk/`.

- `sdk/go/`: Go SDK-facing examples and usage notes
- `sdk/python/`: Python package, examples, tests, and wrapper integrations

The exported Go client library itself remains in `pkg/cmdagentclient`, which is the conventional location for reusable Go packages. The `sdk/go/` area is reserved for SDK-facing examples and related materials.

Current SDK documentation covers:

- starting argv commands, shell commands, and shell sessions
- listing and retrieving execution and transfer metadata
- uploading and downloading files and archives
- deleting one finished history item
- clearing finished history for the authenticated identity
