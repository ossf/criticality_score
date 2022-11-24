# Infrastructure

This directory contains various infrastructure related components.

## Contents

- **cloudbuild**: various Google Cloud Build configurations for automated
  building container images and kicking off new releases in Google Cloud Deploy.
- **envs**: Kubernetes and Kustomize based configurations for staging and prod
  environments.
  - **base**: shared configuration for prod and staging
  - **prod**: production specific configuration
  - **staging**: staging specific configuration
- **k8s**: Misc Kubernetes configurations for Pods that live outside the staging
  and prod environments.
- **test**: A Docker Compose version of the infrastructure for local testing and
  validation.
- **clouddeploy.yaml**, **skaffold.yaml**: Google Cloud Deploy configuration for
  managing the release process of the Criticality Score infrastructure.

## Deploy Process

1. When a new commit is made to `main` (e.g. merged PR):
    - `collect-signals` container image is built and pushed.
    - `csv-transfer` container image is built and pushed.
    - `enumerate-github` container image is built and pushed.
1. Every weekday a scheduled trigger starts a Google Cloud Build process (see
   [release.yaml](https://github.com/ossf/criticality_score/blob/main/infra/cloudbuild/release.yaml)).
    1. `collect-signals` and `csv-transfer` container images are pulled for the
       current commit SHA, ensuring the container images are present.
    1. A new Cloud Deploy release is created (if it doesn't exist already).
        - Release named: `rel-${SHORT_SHA}`.
        - Images are tagged with `${$COMMIT_SHA}` and used in the release.
        - Scorecard images are hardcoded to match go.mod.
1. Cloud Deploy automatically releases to the
   [staging environment](https://github.com/ossf/criticality_score/blob/main/infra/envs/staging).
    - The staging environment runs a short run each weekday.
1. Once a staging release is confirmed as working, the release can be promoted
   to the [production environment](https://github.com/ossf/criticality_score/blob/main/infra/envs/prod).
    - Ideally this should be done between executions to avoid version skew
      issues.

## Cheat Sheet

### Skaffold and Kustomize Output Inspection

To inspect the expanded Kubernetes configuration for each environment use the
following commands, replacing the environment with the relevant one.

For Kustomize (fast):

```shell
kubectl kustomize ./infra/envs/{staging,prod}
```

For Skaffold :

```shell
cd infra && \
  skaffold render -f ./skaffold.yaml --offline -p {staging,prod}
```

### Kubernetes Information

Connecting to the cluster with `gcloud`

```shell
gcloud container clusters get-credentials --region us-central1-c criticality-score
```

Verify context:

```shell
kubectl config current-context
```

#### Managing Secrets

Updating Production GitHub access tokens:

```shell
kubectl create secret generic github --from-literal=token=$GITHUB_AUTH_TOKENS
```

Updating Staging GitHub access tokens:

```shell
kubectl create secret generic github-staging --from-literal=token=$GITHUB_AUTH_TOKENS
```

**Note:** `github` and `github-staging` must be disjoint sets of GitHub
personal access tokens. If they share tokens one environment may starve the
other.
