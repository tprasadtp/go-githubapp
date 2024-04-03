# Example [![go-reference](https://img.shields.io/badge/godoc-reference-5272b4?labelColor=3a3a3a&logo=go&logoColor=959da5)](https://pkg.go.dev/github.com/tprasadtp/go-githubapp/examples/app-token)

An example program to obtain installation access token for an app.

> [!IMPORTANT]
>
> This is a minimal _example_ and is **NOT** covered by semver compatibility guarantees.
> Use [gh-app-token] for a stable CLI which also supports keys stored in KMS and various
> PKCS formats.

## Usage

```
Tool to obtain installation access token or JWT for a Github App

This is a simple example CLI and is not covered by semver compatibility guarantees.
Use https://github.com/tprasadtp/gh-app-token if you need a CLI.

Usage: go run github.com/tprasadtp/go-githubapp/examples/app-token@latest

Flags:
  -app-id uint
    	GitHub app ID (required)
  -format string
    	Output format template
  -installation-id uint
    	Installation ID
  -owner string
    	Installation owner
  -private-key string
    	Path to PKCS1 private key file (required)
  -repos string
    	Comma separated list of repositories
  -revoke
    	Revoke all tokens provided
```

## Example Usage

To obtain installation access token for all the repos run the following.

```
go run github.com/tprasadtp/go-githubapp/examples/app-token@latest \
    -app-id <app-id> \
    -private-key <key-file.pem> \
    -owner <installation-owner>
```

Should return something like,

```
Token             : ghs_xxxxxxxxxxxxxxx
Owner             : gh-integration-tests
Installation      : 1234567
Repositories      : []
Permissions       : map[contents:read issues:read metadata:read]
BotUsername       : gh-integration-tests-app[bot]
BotCommitterEmail : 98765432+gh-integration-tests-app[bot]@users.noreply.github.com
```

where `ghs_xxxx`is installation token which can be used for API and git operations.

[gh-app-token]: https://github.com/tprasadtp/gh-app-token
