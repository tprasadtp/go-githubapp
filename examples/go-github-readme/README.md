# Example

An example program to get readme file for a repository using [google/go-github].

> **Warning**
>
> - This is minimal _example_ and is **NOT** covered by semver compatibility guarantees.
>   Use [gh-app-token] for a stable CLI which also supports keys stored in KMS and various
>   PKCS formats.
> - This example is its own module and if you make changes to `go-githubapp` it may not
>   be reflected without using go workspaces or replace directive.

## Usage

```
CLI to get README for a repository

This is an example CLI and is not covered by semver compatibility guarantees.

Flags:
  -app-id uint
        GitHub app ID (required)
  -private-key string
        Path to PKCS1 private key file (required)
  -repo string
        Repository in {owner}/{repository} format (required)
```

## Example Usage

To obtain README from a private repository accessible to installation,

```
go run example.go \
    -app-id <app-id> \
    -private-key private-key.pem \
    -repo {owner}/{repo}
```

Should return something like,

```
gh-integration-tests/go-githubapp-repo-one README:

# About This Repository

This repository is used for integration tests for [github.com/tprasadtp/go-githubapp].

[github.com/tprasadtp/go-githubapp]: https://github.com/tprasadtp/go-githubapp
```

[google/go-github]: github.com/google/go-github
