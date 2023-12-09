# Integration testing infra

- Create a new github organization.
- Enroll your newly created organization to allow fine-grained access tokens.
[See this for more info](https://docs.github.com/en/organizations/managing-programmatic-access-to-your-organization/setting-a-personal-access-token-policy-for-your-organization).
- Create a short-lived GitHub access token under the organization which has following
repository permissions.
    - `administration:write` (["Administration"][administration:write])
    - `contents:write` (["Contents"][contents:write])
    - `workflow:write` (["Workflows"][workflow:write])
- Set organization name as `GITHUB_OWNER` env variable.
- Set above obtained token as `GITHUB_TOKEN` env variable.
- Create/Select a pulumi stack.
    ```
    pulumi stack select default
    ```
- Create resources.
    ```
    pulumi up
    ```

- Create a new github app under the organization created. See [this](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app) for more info.
- Created app **MUST** have the following permissions. Avoid assigning write permissions.
    - Metadata -> ReadOny
    - Contents -> ReadOnly
- `go-githubapp-repo-one`, `go-githubapp-repo-two` and `go-githubapp-repo-no-access`
**MUST** be private.
- Install the newly created app and grant it access to **ONLY** `go-githubapp-repo-one` and
`go-githubapp-repo-two` repositories. It **MUST NOT** have access to `go-githubapp-repo-no-access`.
- Make a note of App ID.
- Create a new app private key and download it.

[administration:write]:https://docs.github.com/en/rest/overview/permissions-required-for-github-apps?apiVersion=2022-11-28#repository-permissions-for-administration
[contents:write]: https://docs.github.com/en/rest/overview/permissions-required-for-github-apps?apiVersion=2022-11-28#repository-permissions-for-contents
[workflow:write]:https://docs.github.com/en/rest/overview/permissions-required-for-github-apps?apiVersion=2022-11-28#repository-permissions-for-workflows
