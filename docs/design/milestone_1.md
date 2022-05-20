
# Criticality Score Revamp: Milestone 1

- Author: [calebbrown@google.com](mailto:calebbrown@google.com)
- Updated: 2022-04-29

## Goal

Anyone can reliably generate the existing set of signal data using the
`criticality_score` GitHub project, and calculate the scores using the
existing algorithm.

Additionally there will be a focus on supporting future moves towards scaling
and automating criticality score.

For this milestone, collecting dependent signal data sourced from
[deps.dev](https://deps.dev) will also be added to improve the overall
quality of the score produced.

### Non-goals

**Improve how the score is calculated.**

While this is overall vital, the ability to calculate the score depends on
having reliable signals to base the score on.

**Cover source repositories hosted on non-GitHub hosts.**

Critical projects are hosted on GitLab, Bitbucket, or even self-hosted. These
should be supported, but given that over 90% of open source projects are
hosted by GitHub it seems prudent to focus efforts there first.

**De-dupe mirrors from origin source repositories.**

Mirrors are frequently used to provide broader access to a project. Usually
when a self-hosted project uses a public service, such as GitHub, to host a
mirror of the project.

This milestone will not attempt to detect and canonicalize mirrors.

## Background

The OpenSSF has a
[Working Group (WG) focused on Securing Critical Projects](https://github.com/ossf/wg-securing-critical-projects).
A key part of this WG is focused on determining which Open Source projects are
"critical". Critical Open Source projects are those which are broadly depended
on by organizations, and present a security risk to those organizations, and
their customers, if they are not supported.

This project is one of a small set of sources of data used to find theses
critical projects.

The current Python implementation available in this repo has been stagnant for
a while.

It has some serious problems with how it enumerates projects on GitHub (see
[#33](https://github.com/ossf/criticality_score/issues/33)), and lacks robust
support for non-GitHub projects (see
[#29](https://github.com/ossf/criticality_score/issues/29)).

There are problems with the existing signals being collected (see
[#55](https://github.com/ossf/criticality_score/issues/55),
[#102](https://github.com/ossf/criticality_score/issues/102)) and interest in
exploring other signals and approaches
([#53](https://github.com/ossf/criticality_score/issues/53),
[#102](https://github.com/ossf/criticality_score/issues/102) deps.dev,
[#31](https://github.com/ossf/criticality_score/issues/31),
[#82](https://github.com/ossf/criticality_score/issues/82), etc).

Additionally, in [#102](https://github.com/ossf/criticality_score/issues/102) I propose an approach to improving the quality of the criticality score.

## Design Overview

This milestone is a fundamental rearchitecturing of the project to meet the
goals of higher reliability, extensibility and ease of use.

The design focuses on:

- reliable GitHub project enumeration.
- reliable signal collection, with better dependent data.
- being able to update the criticality scores and rankings more frequently.

Please see the [glossary](../glossary.md) for a terms used in this project.

### Multi Stage

The design takes a multi stage approach to generating raw criticality signal
data ready for ingestion into a BigQuery table.

The stages are:

* **Project enumeration** - produce a list of project repositories, focusing
  initially on GitHub for Milestone 1.
* **Raw signal collection** - iterate through the list of projects and query
  various data sources for raw signals.
* **BigQuery ingestion** - take the raw signals and import them into a BigQuery
  table for querying and scoring.

Some API efficiency is gained by collecting some raw signals during project
enumeration. However, the ability to run stages separately and at different
frequencies improves the overall reliability of the project, and allows for raw
signal data to be refreshed more frequently.

## Detailed Design

### Project enumeration

#### Direct GitHub Enumeration

##### Challenges

* GitHub has a lot of repos. Over 2.5M repos with 5 or more stars, and over
  400k repos with 50 or more stars at the time of writing.
* GitHub's API only allows you to iterate through 1000 results.
* GitHub's API has limited methods of sorting and filtering.

Given these limitations it is difficult to extract all the repositories over
a certain number of stars, as the number of repositories with low stars exceeds
the 1000 result limit of GitHub's API.

The lowest number of stars that returns fewer than 1000 results can be improved
by stepping through each creation date.

With a sufficiently high minimum star threshold (e.g. 20), most creation dates
will have fewer than 1000 results in total.

##### Algorithm

* Set `MIN_STARS` to a value chosen such that the number of repositories with
  that number of stars is less than 1000 for any given creation date.
* Set `STAR_OVERLAP`, `START_DATE` and `END_DATE`
* For each `DATE` between `START_DATE` and `END_DATE`:
    * Set `MAX_STARS` to infinity
    * Search for repos with a creation date of `DATE` and stars between
      `MAX_STARS` and `MIN_STARS` inclusive, ordered from highest stars to
      lowest.
    * While True:
        * For each repository (GitHub limits this to 1000 results):
            * If the repository has not been seen:
                * Add it to the list of repositories
        * If there were fewer than 1000 results:
            * Break
        * Set `MAX_STARS` to the the number of stars the last repository
          returned + `STAR_OVERLAP`
        * If `MAX_STARS` is the same as the previous value
            * Break

The current implementation of this algorithm has a difference between GitHub
search of less than 0.05% for >=20 stars (GitHub search was checked ~12 hours
after the algorithm finished) and took 4 hours with 1 worker and 1 token.

##### Rate Limits

A pool of GitHub tokens will be supported for increased performance.

A single GitHub token has a limit of "5000" each hour, a single search page
consumes "1", and returning the 1000 results from a search consumes "10". This
allows 500 search queries per hour for a single token.

##### Output

Output from enumeration will be a text file containing a list of GitHub urls.

#### Static Project URL Lists

Rather than repeatedly query project repositories for a list of projects, use
pre-generated static lists of project repository URLs.

Sources:

* Prior invocations of the enumeration tool
* Manually curated lists of URLs
* [GHTorrent](https://ghtorrent.org/) data dumps

##### GHTorrent

GHTorrent monitors GitHub's public event feed and provides a fairly
comprehensive source of projects.

Data from GHTorrent needs to be extracted from the SQL dump and filtered to
eliminate deleted repositories.

The 2021-03-06 dump includes approx 190M repositories. This many repositories
would need to be curated to ensure each repository is still available. Culling
for "interesting" (e.g. more than 1 star) repositories may also be useful to
limit the amount of work generating signals.

#### Future Sources of Projects

There are many other sources of projects for future milestones that can be
used. These are out-of-scope for Milestone 1, but worth listing.

* Other source repositories such as GitLab and Bitbucket.
* [https://deps.dev/](https://deps.dev/) projects. This source captures many
  projects that exist in package repositories and helps connect projects to
  their packages and dependents.
* GHTorrent or GH Archive - these can avoid the expense of querying GitHub's
  API directly.
* Google dorking - use Google's advanced search capabilities to find
  self-hosted repositories (e.g. cgit, gitea, etc)
* JIRA, Bugzilla, etc support for issue tracking

### Raw Signal Collection

This stage is when the list of projects are iterated over and for each project
a set of raw signal data is output.

#### Input / Output

Input:

* One or more text files containing a list of project urls, one URL per line

Output:

* Either JSON or CSV formatted records for each project in UTF-8, including
  the project url. The output will support direct loading into BigQuery.

#### Signal Collectors

Signal collection will be built around multiple signal _collectors_ that
produce one or more _signals_ per repository.

Signal collectors fall into one of three categories:

* Source repository and hosting signal collectors (e.g. GitHub, Bitbucket,
  cGit)
* Issue tracking signal collectors (e.g. GitHub, Bugzilla, JIRA)
* Additional signal collectors (e.g deps.dev)

Each repository can have only one set of signals from a source repository
collector and one set of signals from an issue tracking signal collector, but
can have signals from many additional collectors.

#### Repository Object

During the collection process a repository object will be created and passed to
each collector.

As each part of the collection process runs, data will be fetched for a
repository. The repository object will serve as the interface for accessing
repository specific data so that it can be cached and limit the amount of
additional queries that need to be executed.

#### Collection Process

The general process for collecting signals will do the following:

* Initialize all the collectors
* For each repository URL
    * Gather basic data about the repository (e.g. stars, has it moved, urls)
        * It may have been removed, in which case the repository can be
          skipped.
        * It may not be "interesting" (e.g. too few stars) and should be
          skipped.
        * It may have already been processed and should be skipped.
    * Determine the set of collectors that apply to the repository.
    * For each collector:
        * Start collecting the signals for the current repository
    * Wait for all collectors to complete
    * Write the signals to the output.

#### Signal Fields

##### Naming

Signal fields will fall under the general naming pattern of
`[collector].[name]`.

Where `[collector]` and `[name]` are made up of one or more of the
following:

* Lowercase characters
* Numbers
* Underscores

The following restrictions further apply to `[collector]` names:

* Source repository signal collectors must use the `repo` collector name
* Issue tracking signal collectors must use the `issues` collector name
* Signals matching the original set in the Python implementation can also use
  the `legacy` collector name
* Additional collectors can use any other valid name.

Finally, `[name]` names must include the unit value if it is not implied by
the type, and any time constraints.

* e.g. `last_update_days`
* e.g. `comment_count_prev_year`

##### Types

For Milestone 1, all signal fields will be scalars. More complex data types are
out of scope.

Supported scalars can be:

* Boolean
* Int
* Float
* String
* Date
* DateTime

All Dates and DateTimes must be in UTC and will be output in the
RFC3339/ISO8601 format: `YYYY-MM-DDTHH:mm:ssZ`. (See
[`time.RFC3339`](https://pkg.go.dev/time#pkg-constants))

Strings will support Unicode.

Null values for scalars are supported. A null value indicates that the signal
could not be collected. To simplify output parsing, an empty string is
equivalent to a null string and is to be interpreted as a null string.

#### Batching (out of scope)

More efficient usage of GitHub's APIs can be achieved by batching together
related requests. Support for batching is considered out of scope for
Milestone 1.

### BigQuery Ingestion

Injection into BigQuery will be done for Milestone 1 using the `bq` command
line tool.

### Language Choice

The Scorecard project and Criticality Score share many of the same needs.

Scorecards also interacts with the GitHub API, negotiates rate limiting and
handles pools of GitHub tokens.

Therefore it makes sense to move towards these projects sharing code.

As Scorecards is a more mature project, this requires Criticality Score to be
rewritten in Go.
