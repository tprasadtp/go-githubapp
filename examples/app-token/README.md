# Example

An example program to obtain installation access token for an app.

> **Warning**
>
> This is a minimal _example_ and is **NOT** covered by semver compatibility guarantees.
> Use [gh-app-token] for a stable CLI which also supports keys stored in KMS and various
> PKCS formats.

## Usage

```
Tool to obtain installation access token or JWT for a Github App

This is a simple example CLI and is not covered by semver compatibility guarantees.
Use https://github.com/tprasadtp/gh-app-token if you need a CLI.

Usage: go run github.com/tprasadtp/go-githubapp/example@latest

Flags:
  -app-id uint
        GitHub app ID (required)
  -install-id uint
        Installation ID
  -jwt
        Generate JWT
  -owner string
        Installation owner
  -private-key string
        Path to PKCS1 private key file (required)
  -repos string
        Comma separated list of repositories
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
Token        : ghs_xxxxx
Owner        : github-username
Installation : 9999
Permissions  : map[contents:read issues:read metadata:read]
user.name    : gh-integration-tests-app[bot]
user.email   : <app-user-id>+gh-integration-tests-app[bot]@users.noreply.github.com
```

where `ghs_xxxx`is installation token which can be used
for API and git operations.

[gh-app-token]: https://github.com/tprasadtp/gh-app-token
