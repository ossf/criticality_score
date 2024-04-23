# GitHub Enumeration Tool

This tool is used to reliably enumerate projects on GitHub.

The output of this tool is can be used as an input for the `criticality_score`
tool, or for input for the `collect_signals` worker.

## Example

```shell
$ export GITHUB_TOKEN=ghp_x  # Personal Access Token Goes Here 
$ enumerate_github \
    -start 2008-01-01 \
    -min-stars=10 \
    -workers=1 \
    -out=github_projects.txt
```

## Install

```shell
$ go install github.com/ossf/criticality_score/v2/cmd/enumerate_github@latest
```

## Usage

```shell
$ enumerate_github [FLAGS]...
```

The URL for each repository is written to the output. By default `stdout` is used
for output.

`FLAGS` are optional. See below for documentation.

### Authentication

A comma delimited environment variable with one or more GitHub Personal Access
Tokens must be set

Supported environment variables are `GITHUB_AUTH_TOKEN`, `GITHUB_TOKEN`, 
`GH_TOKEN`, or `GH_AUTH_TOKEN`.

Example:

```shell
$ export GITHUB_TOKEN=ghp_abc,ghp_123
```

### Flags

#### Output flags

- `-out FILE` specify the `FILE` to use for output. By default `stdout` is used.
- `-append` appends output to `FILE` if it already exists.
- `-force` overwrites `FILE` if it already exists and `-append` is not set.
- `-format {text|scorecard}` indicates the format to use for output. `text` is
  used by default and consists of one URL per line. `scorecard` outputs a CSV
  file compatible with the [scorecard](https://github.com/ossf/scorecard)
  project.

If `FILE` exists and neither `-append` nor `-force` is set the command will fail.

#### Date flags

- `-start date`
        the start date to enumerate back to. Must be at or after `2008-01-01`. Defaults to `2008-01-01`.
- `-end date`
        the end date to enumerate from. Defaults to today's date.

#### Query/Star flags

- `-min-stars int` only enumerates repositories with this or more of stars
  Defaults to `10`.
- `-query string` sets the base query to use for enumeration. Defaults to
  `is:public`. See GitHub's [search help](https://docs.github.com/en/search-github/searching-on-github/searching-for-repositories)
  for more detail.
- `-require-min-stars` abort execution if `-min-stars` can't be reached during
  enumeration. If not set some repositories created on a certain date may not
  be included.
- `-star-overlap int` the number of stars to overlap between queries. Defaults
  to `5`. A an overlap is used to avoid missing repositories whose star count
  changes during enumeration.

#### Misc flags

- `-log level` set the level of logging. Can be `debug`, `info` (default), `warn` or `error`.
- `-workers int` the total number of concurrent workers to use. Default is `1`.
- `-help` displays help text.

## How It Works

Refer to [Milestone 1](../../docs/design/milestone_1.md) for details on the
algorithm.

## Q&A

### Q: What is the lowest practical setting for `-min-stars`

10 has been successfully tested, although lower may be possible.

*TODO* -- more detail

### Q: How long does it take?

A single GitHub Personal Access Token took about 4 hours to return all
projects with >= 20 stars.

Faster performance can be achieved with more Personal Access Tokens and
additional workers.

### Q: How many workers should I use?

Generally, use 1 worker for each Personal Access Token.

More workers than tokens may result in secondary rate limits.

It is possible that more restricted searches will succeed with more workers per
token.

## Development

Rather than installing the binary, use `go run` to run the command.

For example:

```shell
$ go run ./cmd/enumerate_github [FLAGS]...
```

Limiting the data allows for runs to be completed quickly. For example:

```shell
$ go run ./cmd/enumerate_github \
    -log=debug \
    -start=2022-06-14 \
    -end=2022-06-21 \
    -min-stars=20
```
