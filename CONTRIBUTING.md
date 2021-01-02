# Contributing to OSS Criticality Score!

Thank you for contributing your time and expertise to the OSS Criticality Score project.
This document describes the contribution guidelines for the project.

**Note:** Before you start contributing, you must read and abide by our **[Code of Conduct](./CODE_OF_CONDUCT.md)**.

## Contributing code

### Getting started

1.  Create [a GitHub account](https://github.com/join)
1.  Create a [personal access token](https://docs.github.com/en/free-pro-team@latest/developers/apps/about-apps#personal-access-tokens)
1.  Set up your [development environment](#environment-setup)

Then you can [iterate](#iterating).
    
### Environment Setup

You must install these tools:

1.  [`git`](https://help.github.com/articles/set-up-git/): For source control.

1.  [`python`](https://www.python.org/downloads/): For running code.
 
1.  [`python-gitlab`](https://pypi.org/project/python-gitlab/) and [`PyGithub`](https://pypi.org/project/PyGithub/) pip packages.

```shell
pip3 install python-gitlab PyGithub
```

## Iterating

1. Make any code changes to the criticality score algorithm
[here](https://github.com/ossf/criticality_score/tree/main/criticality_score).

1. Run the criticality score code using:

```shell
python3 -m criticality_score.run --repo=<repo_url>
```


