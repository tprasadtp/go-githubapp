# Generating test data

For generating test data, use a non production isolated github organization.

## Generating WebHook test data

- Install gh webhook extension (if required).
    ```console
    gh extension install cli/gh-webhook
    ```

- In another terminal, run test webhook server which dumps raw HTTP requests to file.
    ```console
    go run internal/testdata/webhooks/generate.go \
        -port 8888 \
        -secret fa1286b4-ff70-4cf0-9471-443c796ff13b \
        -dir internal/testdata/webhooks/
    ```

- In another terminal, run webhook forwarder,
    ```console
    gh webhook forward \
        --events="*"  \
        --repo=gh-integration-tests/go-githubapp-repo-one \
        --secret fa1286b4-ff70-4cf0-9471-443c796ff13b \
        --url http://localhost:8888/webhook
    ```

- Generate a github app installation access token which has access to a repo
with issues write permission. Alternatively you can use your own github account for this.

    ```console
    go run ./example/token.go \
        -app-id <app-id> \
        -key <private-key-file-path> \
        -repos <owner/repo>
    ```

- Create some repository events like opening an issue or commenting on an issue.
Use the token generated or default user credentials.

    ```console
    gh issue create --repo {owner/repo} \
        --title "Test issue" \
        --body "test issue"
    ```

- Stop test webhook server and webhook forwarder.
