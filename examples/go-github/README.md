# Example

An example program to get readme file for a repository using [google/go-github].

> **Warning**
>
> - This is minimal _example_ and is **NOT** covered by semver compatibility guarantees.
>   Use [gh-app-token] for a stable CLI which also supports keys stored in KMS and various
>   PKCS formats.
> - This example uses `replace` directive, thus it might be necessary to initialize
>   go workspaces to run this example.


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

```
go run example.go \
    -app-id 394007 \
    -private-key private-key.pem \
    -repo {owner}/{repo}

gh-integration-tests/go-githubapp-repo-one README:

# About This Repository

This repository is used for integration tests for [github.com/tprasadtp/go-githubapp].

[github.com/tprasadtp/go-githubapp]: https://github.com/tprasadtp/go-githubapp
```

[google/go-github]: github.com/google/go-github
