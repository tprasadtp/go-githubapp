# go-githubapp

HTTP Round Tripper to authenticate to GitHub as GitHub app and utilities for WebHook Verification. Supports authenticating with Installation Token and JWT.

[![Go Reference](https://pkg.go.dev/badge/github.com/tprasadtp/go-githubapp)](https://pkg.go.dev/github.com/tprasadtp/go-githubapp)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/tprasadtp/go-githubapp?label=go&logo=go&logoColor=white)
[![test](https://github.com/tprasadtp/go-githubapp/actions/workflows/test.yml/badge.svg)](https://github.com/tprasadtp/go-githubapp/actions/workflows/test.yml)
[![lint](https://github.com/tprasadtp/go-githubapp/actions/workflows/lint.yml/badge.svg)](https://github.com/tprasadtp/go-githubapp/actions/workflows/lint.yml)
[![License](https://img.shields.io/github/license/tprasadtp/go-githubapp)](https://github.com/tprasadtp/go-githubapp/blob/master/LICENSE)
![release-tag](https://img.shields.io/github/v/tag/tprasadtp/go-githubapp?color=7f50a6&label=release&logo=semver&sort=semver)

> **Note**
>
> For API documentation and examples,
> See https://pkg.go.dev/github.com/tprasadtp/go-githubapp.

- This library is designed to provide automatic authentication for [google/go-github], [github.com/shurcooL/githubv4] or your own HTTP client.
- [Transport] implements [http.RoundTripper] which can authenticate transparently.
It _will_ override `Authorization` header. None of the other headers are modified. Thus,
It is user's responsibility to set appropriate headers as required.

## RoundTripper Example

```go
package main

import "github.com/tprasadtp/go-githubapp"

func main() {
	rt, err := githubapp.NewTransport(ctx, appID, signer,
        githubapp.WithOwner("username"),
        githubapp.WithRepositories("repo-one", "repo-two"),
        githubapp.WithPermissions("contents:read")
    )

    client := &http.Client{
        Transport: rt,
    }

    resp, err := client.Get("/repos/<username>/<repo>/readme")
    // Handle error
    if err != nil {
        panic(err)
    }

    doStuff....
}
```

For full working programs see examples directory.

### AppID

App ID can be found in

Settings -> Developer -> settings -> GitHub App -> About item.

Be sure to select correct organization if you are a member of multiple organizations.

### Private key

This library delegates JWT signing to type implementing [crypto.Signer] interface.
Thus it _may_ be backed by KMS/TPM or other secure key store. Optionally
[github.com/tprasadtp/cryptokms] can be used.

### Installation ID

Typically extracted from webhook request headers. If using [VerifyWebHookRequest],
returned [WebHook] includes `InstallationID`. This is not required if owner is already
specified.

### Limit Permissions of Tokens

[WithPermissions] can be used to limit permissions on the created tokens.
[WithPermissions] accepts permissions in `<scope>:<level>` format.
Please check with GitHub API documentation on supported scopes. Requested
permissions cannot permissions existing on the _installation_.

### Limit Scope of Tokens to set of Repositories

[WithRepositories] can be used to limit scope of created access tokens to list of
repositories specified. Repositories MUST belong to a single installation i.e MUST have
a single owner. This accepts repositories in `{owner}/{repo}` format or just name of the
repository. If only name is specified, then it **MUST** be used with [WithOwner] or
[WithInstallationID].

### Using GitHub Enterprise Server

[WithEndpoint] can be used to use custom GitHub REST endpoint. This endpoint will
**ONLY** be used for token renewals and verifying app installation and not http client using
the [Transport].


## Authenticating as an App (JWT)

When none of the installation options [WithOwner], [WithInstallationID] or [WithRepositories]
are specified, [Transport] authenticates as an app. Some API endpoints like listing
installations are only accessible to app.

## App Webhooks

[VerifyWebHookRequest] provides a way to verify webhook payload and extract event data from
headers. See API docs for more info.

[google/go-github]: https://github.com/google/go-github
[github.com/shurcooL/githubv4]: https://github.com/shurcooL/githubv4
[github.com/tprasadtp/cryptokms]: https://github.com/tprasadtp/cryptokms

[http.RoundTripper]: https://pkg.go.dev/net/http#RoundTripper
[crypto.Signer]: https://pkg.go.dev/crypto#Signer
[VerifyWebHookRequest]: https://pkg.go.dev/github.com/tprasadtp/go-githubapp#VerifyWebHookRequest
[WithRepositories]: https://pkg.go.dev/github.com/tprasadtp/go-githubapp#WithRepositories
[WithInstallationID]: https://pkg.go.dev/github.com/tprasadtp/go-githubapp#WithInstallationID
[WithInstallationID]: https://pkg.go.dev/github.com/tprasadtp/go-githubapp#WithInstallationID
[WithOwner]: https://pkg.go.dev/github.com/tprasadtp/go-githubapp#WithOwner
[WithPermissions]: https://pkg.go.dev/github.com/tprasadtp/go-githubapp#WithPermissions
[WithEndpoint]: https://pkg.go.dev/github.com/tprasadtp/go-githubapp#WithEndpoint
[Transport]: https://pkg.go.dev/github.com/tprasadtp/go-githubapp#Transport
[WebHook]: https://pkg.go.dev/github.com/tprasadtp/go-githubapp#WebHook
