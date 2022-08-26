# Test infrastructure using docker-compose

This infrastructure is used during development, for local execution, or to
verify the various components work together in a cloud environment.

## Requirements

- Docker and Docker Compose are installed
  ([docs](https://docs.docker.com/compose/install/))
- Docker images have been built. To build them, run:

```shell
$ make build/docker
```

- Environment variable `GITHUB_TOKEN` set with a GitHub personal access token.

## Usage

To start in daemon mode, run:

```shell
$ docker-compose up -d
```

To stop, run:

```shell
$ docker-compose down
```

To view logs, run:

```shell
$ docker-compose logs -f
```

To restart enumerating github, run:

```
$ docker-compose start enumerate-github
```

To access the storage bucket, visit http://localhost:9000/ in a browser, with
username `minio` and password `minio123`.

## Notes

By default it only enumerates GitHub repositories with 100 or more stars, but
this can be set using the `CRITICALITY_SCORE_STARS_MIN` environment variable.
