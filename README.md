# Open Source Project Criticality Score

This project is maintained by members of the
[Securing Critical Projects WG](https://github.com/ossf/wg-securing-critical-projects).

## Goals
1. Generate a **criticality score** for every open source project.

1. Create a list of critical projects that the open source community depends on.

1. Use this data to proactively improve the security posture of these critical projects.

## Criticality Score

A project's criticality score is a number between
**0 (least-critical)** and **1 (most-critical)**. It is based on the following
[algorithm](https://github.com/ossf/criticality_score/blob/main/Quantifying_criticality_algorithm.pdf)
by [Rob Pike](https://github.com/robpike):

<img src="https://raw.githubusercontent.com/ossf/criticality_score/main/images/formula.png" width="359" height="96">

We use the following parameters to derive the criticality score for an
open source project:

| Parameter (S<sub>i</sub>)  | Weight (&alpha;<sub>i</sub>) | Max threshold (T<sub>i</sub>) | Description |
|---|---:|---:|---|
| created_since | 1 | 120 | Time since the project was created (in months) |
| updated_since  | -1 | 120 | Time since the project was last updated (in months) |
| contributor_count | 2 | 5000 | Count of project contributors (with commits) |
| org_count | 1 | 10 | Count of distinct organizations that contributors belong to |
| commit_frequency | 1 | 1000 | Average number of commits per week in the last year |
| recent_releases_count | 0.5 | 26 | Number of releases in the last year |
| closed_issues_count | 0.5 | 5000 | Number of issues closed in the last 90 days |
| updated_issues_count | 0.5 | 5000 | Number of issues updated in the last 90 days |
| comment_frequency | 1 | 15 | Average number of comments per issue in the last 90 days |
| dependents_count | 2 | 500000 | Number of project mentions in the commit messages |

## Usage

The program only requires one argument to run, the name of the repo:

```shell
$ pip3 install criticality-score

$ criticality_score --repo github.com/kubernetes/kubernetes
name: kubernetes
url: https://github.com/kubernetes/kubernetes
language: Go
created_since: 79
updated_since: 0
contributor_count: 3664
org_count: 5
commit_frequency: 102.7
recent_releases_count: 76
closed_issues_count: 2906
updated_issues_count: 5136
comment_frequency: 5.7
dependents_count: 407254
criticality_score: 0.9862
```

You can add your own parameters to the criticality score calculation. For
example, you can add internal project usage data to re-adjust the project's
criticality score for your prioritization needs. This can be done by adding
the `--params <param1_value>:<param1_weight>:<param1_max_threshold> ...`
argument on the command line.

### Authentication

Before running criticality score, you need to
[create a GitHub access token](https://docs.github.com/en/free-pro-team@latest/developers/apps/about-apps#personal-access-tokens)
and set it in environment variable `GITHUB_AUTH_TOKEN`.
This helps to avoid the GitHub's
[api rate limits](https://developer.github.com/v3/#rate-limiting)
with unauthenticated requests.

```shell
# For posix platforms, e.g. linux, mac:
export GITHUB_AUTH_TOKEN=<your access token>

# For windows:
set GITHUB_AUTH_TOKEN=<your access token>
```
### Formatting Results

There are three formats currently: `default`, `json`, and `csv`. Others may be added in the future.

These may be specified with the `--format` flag.

## Public Data

If you're only interested in seeing a list of critical projects with their
criticality score, we publish them in `csv` format.

This data is available on Google Cloud Storage and can be downloaded via the
[`gsutil`](https://cloud.google.com/storage/docs/gsutil_install)
command-line tool or the web browser
[here](https://commondatastorage.googleapis.com/ossf-criticality-score/index.html).

```shell
$ gsutil ls gs://ossf-criticality-score/
gs://ossf-criticality-score/c_top_200.csv
gs://ossf-criticality-score/cplusplus_top_200.csv
gs://ossf-criticality-score/java_top_200.csv
gs://ossf-criticality-score/js_top_200.csv
gs://ossf-criticality-score/python_top_200.csv
...

$ gsutil cat gs://ossf-criticality-score/python_top_200.csv
Project,URL,Language,Created since (months),Updated since (months),Contributors,Orgs for Top15 contributors,Commit freq/week (last yr),Releases (last yr),Closed issues (last 90d),Updated issues (last 90d),Comment freq/issue (last 90d),Commit mentions,Criticality Score
salt,https://github.com/saltstack/salt,Python,119,0,3631,7,65.3,18,861,1713,1.2,20953,0.87988
core,https://github.com/home-assistant/core,Python,87,0,2487,9,168.9,202,4289,5780,3.7,341,0.87196
pandas,https://github.com/pandas-dev/pandas,Python,125,0,2509,7,77.9,13,2341,3454,2.4,3572,0.86588
scikit-learn,https://github.com/scikit-learn/scikit-learn,Python,125,0,2090,8,27.5,6,708,1260,2.4,30453,0.86011
numpy,https://github.com/numpy/numpy,Python,124,0,1211,9,38.4,16,712,1032,3.3,8543,0.8574
...
```

## Contributing

If you want to get involved or have ideas you'd like to chat about, we discuss this project in the [Securing Critical Projects WG](https://github.com/ossf/wg-securing-critical-projects) meetings.

See the [Community Calendar](https://calendar.google.com/calendar?cid=czYzdm9lZmhwNWk5cGZsdGI1cTY3bmdwZXNAZ3JvdXAuY2FsZW5kYXIuZ29vZ2xlLmNvbQ) for the schedule and meeting invitations.

See the [Contributing](CONTRIBUTING.md) documentation for guidance on how to contribute.
