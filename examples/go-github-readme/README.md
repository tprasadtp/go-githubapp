# Example

An example program to get readme file for a repository using [google/go-github].

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
