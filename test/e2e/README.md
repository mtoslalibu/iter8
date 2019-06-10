# End to End iter8 tests

## Running locally

```sh
./test/e2e/e2e.sh -s
```

## Publishing docker images

You need [Ko](https://github.com/google/ko). Make sure to set `$KO_DOCKER_REPO` to `docker.io/villardl` (See issue #34).

```sh
./test/docker-publish-images.sh
```
