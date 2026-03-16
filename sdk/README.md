# SDK Layout

Language-specific SDK assets live under `sdk/`.

- `sdk/go/`: Go SDK-facing examples and usage notes
- `sdk/python/`: Python package, examples, tests, and wrapper integrations

The exported Go client library itself remains in `pkg/cmdagentclient`, which is the conventional location for reusable Go packages. The `sdk/go/` area is reserved for SDK-facing examples and related materials.
