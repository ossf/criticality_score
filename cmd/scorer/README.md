# Scoring Tool

This tool is used to calculate a criticality score based on raw signals
collected across a set of projects.

The input of this tool is usually the output of the `collect_signals` tool.

## Example

```shell
$ scorer \
    -config config/scorer/original_pike.yml \
    -out=scored_signals.txt \
    raw_signals.txt
```

## Install

```shell
$ go install github.com/ossf/criticality_score/v2/cmd/scorer@latest
```

## Usage

```shell
$ scorer [FLAGS]... IN_FILE
```

Raw signals are read as CSV from `IN_FILE`. If `-` is passed in for `IN_FILE`
raw signal data will read from STDIN rather than a file.

Results are re-written in CSV format to the output in descending score order.
By default `stdout` is used for output.

The `-config` flag is required. All other `FLAGS` are optional.
See below for documentation.

### Flags

#### Output flags

- `-out FILE` specify the `FILE` to use for output. By default `stdout` is used.
- `-append` appends output to `OUT_FILE` if it already exists.
- `-force` overwrites `OUT_FILE` if it already exists and `-append` is not set.

If `OUT_FILE` exists and neither `-append` nor `-force` is set the command will
fail.

#### Scoring flags

- `-config string` the name of a YAML config file to use for calculating the
  score. Defaults to the original set of weights and scores.
- `-column string` the name of the column to store the score in. Defaults to
  the name of the config file with `_score` appended (e.g. `config.yml` becomes
  `config_score`).

#### Misc flags

- `-log level` set the level of logging. Can be `debug`, `info` (default),
  `warn` or `error`.
- `-help` displays help text.

## Config file format

```yaml
# The algorithm used to combine the inputs together.
# Currently only "weighted_arithmetic_mean" is supported.
algorithm: weighted_arithmetic_mean

# Inputs is an array of fields used as input for the algorithm.
inputs:
    # The name of the field. This corresponds to the column name in the input
    # CSV file.
    # Required.
  - field: namespace.field

    # The weight of the field. A higher weight means the value will have a
    # bigger impact on the score. A weight of "0" means the input will have no
    # impact on the score.
    # Default: 1.0
    weight: 4.2

    # Bounds defines the lower and upper bounds of values for this input, and
    # whether or not larger or smaller values are considered "better" or more
    # critical.
    # Default: unset.
    bounds:
      # The lower bound as a float. Any values lower will be set to this value.
      # Default: 0 (if bounds is set)
      lower: 100
      # The upper bound as a float. Any values higher will be set to this
      # value.
      # Default: 0 (if bounds is set)
      upper: 1000
      # A boolean indicating whether or not a small value is considered
      # "better" or more critical. Can be "yes", or "no"
      # Default: "no" or "false" (if bounds is set)
      smaller_is_better: no

    # Condition will only include this input when calculating the score if and
    # only if the condition returns true. Only the existance, or non-existance
    # of value in another field can be tested currently.
    # Only one key can be set under `condition` at a time.
    # Default: unset (always true)
    condition:
      # Returns true if the specified field has a value. If the field is does
      # not exist in the input CSV data the result will always be false.
      # Must be used on its own.
      field_exists: namespace.field2

      # Not negates the condition. So a true value becomes false, and a false
      # value becomes true.
      # Must be used on its own.
      not:
        field_exists: namespace.field3

    # The distribution is used to specify the type of statistical distribution
    # for this field of data. This is used to help normalize the data so that
    # it can be better combined. Valid values are "normal", "zipfian".
    # Default: "normal"
    distribution: normal
```

See
[config/scorer](https://github.com/ossf/criticality_score/tree/main/config/scorer)
for examples.

## Development

Rather than installing the binary, use `go run` to run the command.

For example:

```shell
$ go run ./cmd/scorer [FLAGS]... IN_FILE
```

Use STDIN and STDOUT on a subset of data for fast iteration. For example:

```shell
$ head -10 raw_signals.csv | go run ./cmd/scorer \
    -config config/scorer/original_pike.yml \
    -
```

Here is an example of raw signals for `scorer` input:

```csv
repo.url,repo.language,repo.license,repo.star_count,repo.created_at,repo.updated_at,legacy.created_since,legacy.updated_since,legacy.contributor_count,legacy.org_count,legacy.commit_frequency,legacy.recent_release_count,legacy.updated_issues_count,legacy.closed_issues_count,legacy.issue_comment_frequency,legacy.github_mention_count,depsdev.dependent_count,default_score,collection_date
https://github.com/ashawkey/RAD-NeRF,Python,MIT License,64,2022-11-23T02:51:07Z,2022-11-23T04:32:02Z,0,0,1,1,0.1,0,0,0,0,0,,0.13958,2022-11-28T02:47:06Z
https://github.com/0DeFi/Solana-NFT-Minting,Python,,19,2022-11-23T17:55:35Z,2022-11-23T18:00:21Z,0,0,1,1,0.06,0,0,0,0,0,,0.13907,2022-11-28T02:47:06Z
https://github.com/0DeFi/Cardano-NFT-Minting,Python,MIT License,19,2022-11-23T17:52:59Z,2022-11-23T18:02:20Z,0,0,1,1,0.1,0,0,0,0,0,,0.13958,2022-11-28T02:47:06Z
https://github.com/0DeFi/MagicEden-Minting-Bot,Python,MIT License,19,2022-11-23T17:58:09Z,2022-11-23T17:59:23Z,0,0,1,1,0.06,0,0,0,0,0,,0.13907,2022-11-28T02:47:06Z
https://github.com/trungdq88/Awesome-Black-Friday-Cyber-Monday,,,1414,2022-11-22T06:16:23Z,2022-11-26T17:42:31Z,0,0,274,7,17.19,0,356,344,0.19,2,,0.43088,2022-11-28T02:47:06Z
https://github.com/HuolalaTech/hll-wp-glog,PHP,Apache License 2.0,99,2022-11-22T02:37:31Z,2022-11-22T17:22:42Z,0,0,1,0,0.06,1,2,0,0,0,,0.12770,2022-11-28T02:47:06Z
https://github.com/flo-at/rustsnake,Rust,MIT License,76,2022-11-14T23:28:32Z,2022-11-25T23:33:43Z,0,0,2,1,0.44,0,2,2,2,0,,0.20238,2022-11-28T02:47:06Z
https://github.com/SeeFlowerX/estrace,Go,MIT License,68,2022-11-22T09:26:34Z,2022-11-24T03:50:27Z,0,0,1,0,0.1,2,1,1,1,0,,0.15949,2022-11-28T02:47:06Z
https://github.com/proofxyz/solidify,Go,MIT License,74,2022-11-22T16:16:23Z,2022-11-22T16:40:06Z,0,0,1,1,0.04,1,4,1,1,0,,0.18551,2022-11-28T02:47:06Z
https://github.com/caidukai/sms-interception,Swift,MIT License,54,2022-11-22T04:47:22Z,2022-11-22T10:03:20Z,0,0,1,0,0.21,0,1,1,0,0,,0.12112,2022-11-28T02:47:06Z
https://github.com/WeilunWang/SinDiffusion,Python,Apache License 2.0,88,2022-11-22T03:52:52Z,2022-11-24T09:03:19Z,0,0,1,1,0.33,0,2,1,0,0,,0.15222,2022-11-28T02:47:06Z
https://github.com/adobe-research/convmelspec,Python,Apache License 2.0,72,2022-11-22T17:29:53Z,2022-11-22T17:56:50Z,0,0,2,1,0.08,1,0,0,0,0,,0.15841,2022-11-28T02:47:06Z
https://github.com/avuenja/tabnews-app,Dart,MIT License,54,2022-11-22T00:13:25Z,2022-11-27T19:03:14Z,0,0,2,2,1.02,5,33,25,0.97,0,,0.26024,2022-11-28T02:47:06Z
https://github.com/Sheng-T/FedMGD,Python,Other,33,2022-11-22T01:51:32Z,2022-11-22T12:52:14Z,0,0,2,0,0.27,0,0,0,0,0,,0.12310,2022-11-28T02:47:06Z
https://github.com/bmarsh9/gapps,HTML,Other,27,2022-11-22T00:04:32Z,2022-11-23T23:32:21Z,0,0,2,0,0.23,0,10,0,0.4,1,,0.15769,2022-11-28T02:47:06Z
https://github.com/ankane/polars-ruby,Ruby,MIT License,57,2022-11-22T05:59:43Z,2022-11-28T02:09:26Z,0,0,1,0,4.44,292,1,0,0,0,,0.18558,2022-11-28T02:47:06Z
https://github.com/adnanali-in/cfi_b22_classwork,JavaScript,,23,2022-11-22T04:51:23Z,2022-11-25T04:32:01Z,0,0,1,1,0.1,0,0,0,0,0,,0.13958,2022-11-28T02:47:06Z
https://github.com/georgetomzaridis/aade-publicity-search-js,JavaScript,MIT License,23,2022-11-22T01:29:45Z,2022-11-23T13:35:27Z,0,0,2,2,0.33,0,3,3,0.33,1,,0.20273,2022-11-28T02:47:06Z
https://github.com/mobcode1337/Twitter-Account-Creator,Python,,19,2022-11-22T12:58:49Z,2022-11-23T00:10:02Z,0,0,1,0,0.04,0,0,0,0,0,,0.11128,2022-11-28T02:47:06Z
```

An example for the raw signal data can be found at: https://commondatastorage.googleapis.com/ossf-criticality-score/index.html?prefix=2022.11.28/024706/
