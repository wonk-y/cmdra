# CmdAgent Docusaurus Site

This site is currently pinned to Docusaurus `3.6.3` so it remains compatible with the repository's Node 18 toolchain.

From `website/`:

```bash
npm install
npm start
npm run build
npm run serve
```

The generated static output is written to `website/build`.

The site documents:

- daemon install and service usage on Linux, macOS, and Windows
- `cmdagentctl` and `cmdagentui` operations, including `delete` and `clear-history`
- Go and Python SDK usage
- Robot Framework and Ansible integration notes

## GitHub Pages

The repository includes a Pages workflow in `.github/workflows/docs-pages.yml`.

It builds the site from `website/`, uploads `website/build`, and deploys through GitHub Pages Actions.

Defaults:

- project pages repo: `https://<owner>.github.io/<repo>/`
- user or organization pages repo: `https://<owner>.github.io/`

Optional GitHub repository variables:

- `DOCS_URL`
- `DOCS_BASE_URL`

Use those when you need a custom domain or a non-default base path.
