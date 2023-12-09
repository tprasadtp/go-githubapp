# Generating test data

For generating test data, use a non-production, isolated GitHub organization.
See [../testinfra/README.md](../testinfra/README.md) for creating required
resources.

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
        --repo=<sandbox-org>/go-githubapp-repo-one \
        --secret fa1286b4-ff70-4cf0-9471-443c796ff13b \
        --url http://localhost:8888/webhook
    ```

- Create some repository events like opening an issue or commenting on an issue.
Optionally, use installation access token which has appropriate permissions.

    ```console
    gh issue create --repo <sandbox-org>/go-githubapp-repo-one \
        --title "Test issue" \
        --body "test issue"
    ```

- Stop both test webhook server and webhook-forwarder.
