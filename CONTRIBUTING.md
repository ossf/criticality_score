# Contributing to OSS Criticality Score!

Thank you for contributing your time and expertise to the OSS Criticality Score
project. This document describes the contribution guidelines for the project.

**Note:** Before you start contributing, you must read and abide by our
**[Code of Conduct](./CODE_OF_CONDUCT.md)**.

## Contributing code

### Getting started

1.  Create [a GitHub account](https://github.com/join)
1.  Create a [personal access token](https://docs.github.com/en/free-pro-team@latest/developers/apps/about-apps#personal-access-tokens)
1.  (Optionally) a Google Cloud Platform account for [deps.dev](https://deps.dev) data
1.  Set up your [development environment](#environment-setup)

Then you can [iterate](#iterating).

## Environment Setup

You must install these tools:

1.  [`git`](https://help.github.com/articles/set-up-git/): For source control.

1.  [`go`](https://go.dev/dl/): For running code.

And optionally:

1.  [`gcloud`](https://cloud.google.com/sdk/docs/install): For Google Cloud Platform access for deps.dev data.

Then clone the repository, e.g:

```shell
$ git clone git@github.com:ossf/criticality_score.git
$ cd criticality_score
```

## Iterating

1. Find or create an [issue](https://github.com/ossf/criticality_score/issues)

1. Make code changes to:
    - the [collect_signals CLI tool](https://github.com/ossf/criticality_score/tree/main/cmd/collect_signals)
    - the [GitHub enumerator](https://github.com/ossf/criticality_score/tree/main/cmd/enumerate_github)
    - the [signal collector worker](https://github.com/ossf/criticality_score/tree/main/cmd/collect_signals)
    - the [scorer](https://github.com/ossf/criticality_score/tree/main/cmd/scorer)
    - the scorer [algorithm configuration](https://github.com/ossf/criticality_score/tree/main/config/scorer)

1. Run your changes. For example, for a single repository this can be done by
   executing:

```shell
$ export GITHUB_TOKEN=ghp_x  # the personal access token created above
$ go run ./cmd/criticality_score \
    -log=debug \
    -depsdev-disable \  # remove if you have a GCP account configured
    "https://github.com/{ a repo }"
```
Note: Each of the tools listed above can be run individually and has their own
README.

4. Ensure your code passes tests and lint checks:

```shell
$ make test
$ make lint
```

5. Commit your change. Upload to a fork, and create a pull request!