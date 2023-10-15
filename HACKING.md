# Development

> **Note**
>
> Testing code, scripts and configuration are _not_ covered by semver compatibility guarantees.

You will need go toolchain 1.21 or later and optionally `pulumi`` and `gh` cli.

## Creating integration test resources

See [./internal/testinfra/README.md](./internal/testinfra/README.md).

## Creating test data

See [./internal/testdata/README.md](./internal/testdata/README.md).

## Integration tests

`go test` will automatically run integration tests if _all_ the following environment variables are set and it can connect to `GO_GITHUBAPP_TEST_BASE_URL`.

| Environment Variable |  Description |
| ---|---
| `GO_GITHUBAPP_TEST_BASE_URL` | Github API endpoint. Defaults to `https://api.github.com` if not set.
| `GO_GITHUBAPP_TEST_OWNER` | Organization name to be used _exclusively_ for testing.
| `GO_GITHUBAPP_TEST_APP_ID` | GitHub app of the app to be used _exclusively_ for testing.
| `GO_GITHUBAPP_TEST_APP_PRIVATE_KEY` | GitHub app's private key. __MUST__ be in PEM encoded PKCS1 format.

> **Warning**
>
> - Invalid `GO_GITHUBAPP_TEST_BASE_URL` URL is considered test failure.
> - Integration tests will be skipped if `GO_GITHUBAPP_TEST_BASE_URL` returns server errors.
