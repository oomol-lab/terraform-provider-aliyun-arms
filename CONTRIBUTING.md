# Contributing

Contributions are welcome through GitHub issues and pull requests.

Before submitting a change, run:

```shell
make fmt
make test
make build
make generate
```

Generated documentation under `docs/` must be committed with schema or example
changes. Acceptance tests that modify Alibaba Cloud resources must be clearly
identified and must never rely on committed credentials.
