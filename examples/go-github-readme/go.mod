module github.com/tprasadtp/go-githubapp/examples/go-github-readme

go 1.21

require (
	github.com/google/go-github/v61 v61.0.0
	github.com/tprasadtp/go-githubapp v0.0.0-00010101000000-000000000000
)

require github.com/google/go-querystring v1.1.0 // indirect

replace github.com/tprasadtp/go-githubapp => ./../../
