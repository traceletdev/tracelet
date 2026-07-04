# Examples

Ways to get tracelet running against real projects.

## Config examples

[`config-examples/`](./config-examples) — annotated `tracelet.config.json` starting points
(basic, strict, CI). Copy one into your project root and run `npx tracelet lint`.

## Runnable fixture apps

The repository ships minimal, buildable apps used by the integration tests — handy as
end-to-end references:

- [`tests/fixtures/next-basic`](../tests/fixtures/next-basic) — a Next.js project with a
  collected `.tracelet/stats.json`.
- [`tests/fixtures/vite-basic`](../tests/fixtures/vite-basic) — the Vite equivalent.

Each already has a `tracelet.config.json` and stats, so you can point tracelet at them:

```bash
cd tests/fixtures/next-basic
npx tracelet lint
```

See the [Framework Adapters guide](../docs/adapters.md) for wiring tracelet into your own
Next.js or Vite build.
