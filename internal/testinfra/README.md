# Integration Testing infra

- Create a new github organization.
- Enroll your newly created organization to allow fine grained access tokens.
[See this for more info](https://docs.github.com/en/organizations/managing-programmatic-access-to-your-organization/setting-a-personal-access-token-policy-for-your-organization).
- Create a short lived a github access token under the organization which has following
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

[administration:write]:https://docs.github.com/en/rest/overview/permissions-required-for-github-apps?apiVersion=2022-11-28#repository-permissions-for-administration
[contents:write]: https://docs.github.com/en/rest/overview/permissions-required-for-github-apps?apiVersion=2022-11-28#repository-permissions-for-contents
[workflow:write]:https://docs.github.com/en/rest/overview/permissions-required-for-github-apps?apiVersion=2022-11-28#repository-permissions-for-workflows
