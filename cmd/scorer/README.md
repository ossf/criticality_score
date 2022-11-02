# Scoring Tool

This tool is used to calculate a criticality score based on raw signals
collected across a set of projects.

The input of this tool is usually the output of the `collect_signals` tool.

## Example

```shell
$ scorer \
    -config config/scorer/original_pike.yml \
    raw_signals.txt \
    scored_signals.txt
```

## Install

```shell
$ go install github.com/ossf/criticality_score/cmd/scorer
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
$ go run ./cmd/scorer [FLAGS]... IN_FILE OUT_FILE
```

Use STDIN and STDOUT on a subset of data for fast iteration. For example:

```shell
$ head -10 raw_signals.csv | go run ./cmd/scorer \
    -config config/scorer/original_pike.yml \
    - -
```