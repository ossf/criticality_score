# Signal Collector

This tool is used to collect signal data for a set of project repositories for
generating a criticality score.

The input of this tool could by the output of the `enumerate_github` tool.

The output of this tool is used as an input for the `scorer` tool, or for input
into a data analysis system such as BigQuery. 

## Example

```shell
$ export GITHUB_TOKEN=ghp_x  # Personal Access Token Goes Here
$ gcloud login --update-adc  # Sign-in to GCP
$ collect_signals \
    github_projects.txt \
    signals.csv
```

## Install

```shell
$ go install github.com/ossf/criticality_score/cmd/collect_signals
```

## Usage

```shell
$ collect_signals [FLAGS]... IN_FILE OUT_FILE
```

Project repository URLs are read from the specified `IN_FILE`. If `-` is passed
in as an `IN_FILE` URLs will read from STDIN.

Results are written in CSV format to `OUT_FILE`. If `OUT_FILE` is `-` the
results will be written to STDOUT.

`FLAGS` are optional. See below for documentation.

### Authentication

`collect_signals` requires authentication to GitHub, and optionally Google Cloud Platform to run.

#### GitHub Authentication

A comma delimited environment variable with one or more GitHub Personal Access
Tokens must be set

Supported environment variables are `GITHUB_AUTH_TOKEN`, `GITHUB_TOKEN`, 
`GH_TOKEN`, or `GH_AUTH_TOKEN`.

Example:

```shell
$ export GITHUB_TOKEN=ghp_abc,ghp_123
```

#### GCP Authentication

BigQuery access requires the "BigQuery User" (`roles/bigquery.user`) role added
to the account used, or be an "Owner".

##### Option 1: `gcloud login`

This option is useful during development. Run `gcloud login --update-adc` to
login to GCP and prepare application default credentials.

##### Option 2: GCE Service Worker

If running on a GCE instance a service worker will be associated with the
machine.

Simply ensure the service worker is added to the "BigQuery User" role.

##### Option 3: Custom Service Worker

A custom service worker is ideal for limiting access to GCP resources.

One can be created through the console or via `gcloud` on the command line.

For example:

```shell
$ # Create the service worker account
$ gcloud iam service-accounts create [SERVICE-ACCOUNT-ID]
$ # Add the service worker to the "BigQuery User" role
$ gcloud projects add-iam-policy-binding [PROJECT-ID] --member="serviceAccount:[SERVICE-ACCOUNT-ID]@[PROJECT-ID].iam.gserviceaccount.com" --role=roles/bigquery.user
$ # Generate a credentials file for the service worker
$ gcloud iam service-accounts keys create [FILENAME].json --iam-account=[SERVICE-ACCOUNT-ID@[PROJECT-ID].iam.gserviceaccount.com
```

To use the service worker account the json credential file needs to be passed
in through the `GOOGLE_APPLICATION_CREDENTIALS` environment variable.

Example:

```shell
$ export GOOGLE_APPLICATION_CREDENTIALS=[FILENAME].json
```

See more on GCP
[service account docs](https://cloud.google.com/iam/docs/creating-managing-service-accounts).

### Flags

#### Output flags

- `-append` appends output to `FILE` if it already exists.
- `-force` overwrites `FILE` if it already exists and `-append` is not set.

If `FILE` exists and neither `-append` nor `-force` is set the command will fail.

#### Google Cloud Platform flags

- `-gcp-project-id string` the Google Cloud Project ID to use. Auto-detects by default.

#### deps.dev Collection Flags

- `-depsdev-disable` disables the collection of signals from deps.dev.
- `-depsdev-dataset string` the BigQuery dataset name to use. Default is `depsdev_analysis`.

#### Misc flags

- `-log level` set the level of logging. Can be `debug`, `info` (default), `warn` or `error`.
- `-help` displays help text.

## Q&A

### Q: How long does it take?

It takes ~2.5 seconds per repository on a fast computer with excellent internet
access.

From experience, if no rate limits are hit, a single worker can collect about
1400 repositories in an hour.

### Q: How many workers should I use?

Generally, use 1 worker per one or two Personal Access Tokens.

On a fast computer with excellent internet access, a single worker can consume
the quota for a single token in about 30-40 minutes.

### Q: Any tips on making it run fast and reliable?

1. Spin up a compute instance in GCE with lots of RAM and fast network:
    - Uses GCP's fast/reliable internet connectivity.
    - Reduces latency and costs for querying BigQuery data (and perhaps 
      GitHub's data too).
   - Prevents downtime due to local IT failures.
1. Shard the input repository list and run multiple instances on different
   hosts with different GitHub tokens:
    - Ensures work progresses even if one instance stops.
    - Provides additional compute and network resources.

### Q: How do I restart after a failure?

If running with a single worker this process is fairly straightforward.

1. Copy the input repository list file to a new file to edit.
1. Open the new file in an editor (note: it may be very large).
1. `tail -25` the output csv file to view the last entries.
1. In the editor, find the entry that corresponds to the last csv entry.
    - If running with a single worker: delete this repository url and *all*
      repository urls above it.
    - If running with multiple workers: manually delete repository urls that
      correspond to entries in the output csv until there are no unprocessed
      repository urls interleaving processed urls. Delete the remaining urls
      above the unprocessed urls.
1. Restart `collect_signals`:
    - Use the new file as the input.
    - Either use a new file as the output, or specify `-append`.

*Note:* when correlating URLs it is possible that the repository has been
renamed.

### Q: How much will GCP usage cost?

deps.dev support is designed to work within the free pricing tier for GCP.

A single BigQuery query of 3Gb data is executed once, with the resulting table
used for subsequent queries.

Network transit costs should be small enough to also sit within the free tier.

A free GCE instance could be used to reduce network transit costs, but may slow
down the collection.

## Development

Rather than installing the binary, use `go run` to run the command.

For example:

```shell
$ go run ./cmd/collect_signals [FLAGS]... IN_FILE... OUT_FILE
```

Pass in a single repo using echo to quickly test signal collection, for example:

```shell
$ echo "https://github.com/django/django" | \
    go run ./cmd/collect_signals \
      -log=debug \
      - -
```