# Example

An example program to obtain installation access token for an app.

> **Warning**
>
> This is minimal _example_ and is **NOT** covered by semver compatibility guarantees.
> Use [gh-app-token] for a stable CLI which also supports keys stored in KMS and various
> PKCS formats.

## Usage

```
Tool to obtain installation access token for a Github App

Usage: go run github.com/tprasadtp/go-githubapp/example@latest

Flags:
  -app-id uint
        GitHub app ID
  -install-id uint
        App installation ID
  -owner string
        Installation owner (username)
  -private-key string
        Path to private key file
  -repos string
        Comma separated list of repositories
```


## Example Usage

To obtain installation access token for all the repos,

```
go run github.com/tprasadtp/go-githubapp/example@latest \
    -app-id <app-id> \
    -private-key <key-file.pem> \
    -owner <github-username>
```

Should returns something like, where `ghs_xxxx`is installation token which can be used
for API and git operations.

```
Token        : ghs_xxxxx
Owner        : github-username
Installation : 000001
Repositories : [go-githubapp-repo-one go-githubapp-repo-two]
Permissions  : map[contents:read issues:read metadata:read]
user.name    : gh-integration-tests-app[bot]
user.email   : <app-user-id>+gh-integration-tests-app[bot]@users.noreply.github.com
```

[gh-app-token]: https://github.com/tprasadtp/gh-app-token
