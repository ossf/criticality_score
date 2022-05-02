# Glossary

This document defines the meaning of various terms used by this project.
This is to ensure they are clearly understood.

Please keep the document sorted alphabetically.

## Terms

### Fork

A _fork_, like a mirror, is a clone or copy of another project's source code or
repository. 

A fork has two primary uses:

* A contributor commiting changes to a fork for preparing pull-requests to the
  main repository.
* A fork may become its own project when the original is unmaintained, or if
  the forker decides to head in a different direction.

Forks merely used for committing changes for a pull-request are not interesting
when calculating criticality scores.

See also "Mirror".

### Mirror

A _mirror_, like a fork, is a clone or copy of another project's source code or
repository. 

A mirror is usually used to provide broader access to a repository, such as
when a self-hosted project mirrors its repository on GitHub.

Mirrors may require de-duping to avoid treating the original repository and
its mirrors as separate projects.

See also "Fork".

### Project

A _project_ is defined as only having a _single repository_, and a _single
issue tracker_. A project may provide multiple _packages_.

There are some "umbrella projects" (e.g. Kubernetes) that have multiple
repositories associated with them, or may use a centralized issue tracker. An
alternative approach would be to treat a project independently of the one or
more repositories that belong to it.

However this approach has the following drawbacks:

* Makes it hard to distinguish between organizations and umbrella projects
* Raises the possibility that a part of the umbrella project that is critical
  to OSS is missed.
* Complicates the calculation required to aggregate signals and generate a
  criticality score.

So instead we define a project as a single repository. This provides a clear
"primary key" we can use for collecting signals.

### Repository

A _repository_ refers to the system used to store and manage access to a
project's source code. Usually a version control system (e.g. git or mercurial)
is used to track and manage changes to the source code.

A _repository_ can be the canonical source of a project's code, or it could
also be a _fork_ or a _mirror_.

A _repository_ is usually owned by an individual or an organization, although
the specifics of how this behaves in practice depends on the repositories host.

### Repository Host

A _repository host_ is the service hosting a _repository_. It may be a service
such as GitHub, GitLab or Bitbucket. It may also be "self-hosted", where the
infrastructure for hosting a repository is managed by the maintainers of a
project.

Self-hosted repositories often deploy an open-source application to provide
access, such as GitLab, cGit, or Gitea.

### Umbrella Project

An _umbrella project_ is a group of related projects that are maintained by a
larger community surrounding the project.

See also "project".