# Config examples

Drop-in `tracelet.config.json` starting points. Copy one to your project root and adjust.

| File | Use case |
|------|----------|
| [`basic.json`](./basic.json) | Sensible defaults for a Next.js app — one error-level budget rule, warnings for the rest. |
| [`strict.json`](./strict.json) | Tight per-route budgets with most rules promoted to `error`; mobile probe with 3 runs. Good for teams that want to hold the line. |
| [`ci.json`](./ci.json) | Vite project tuned for pipelines — an error budget plus a per-route override for a known-heavy page. |

Rule levels are `error` (fails CI), `warn`, or `info`. See the
[Configuration reference](../../docs/config.md) and [Rules catalog](../../docs/rules.md)
for every option.

```bash
cp examples/config-examples/basic.json tracelet.config.json
npx tracelet lint
```
