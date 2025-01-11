# Yato - Go api boilerplate

Boilerplate for Go api. This boilerplate is using the following tools:

- Gin as the http framework
- Gorm as the ORM
- Postgresql as the database
- Rate limit using fixed window algorithm
- Mailer using Sendgrid

# Develop

Clone the project including its submodule

```sh
git clone --recursive https://github.com/SeakMengs/AutoCert.git
```

Development is divided into two environment, `docker` and `local`.

## Docker environment

The docker environment in this project has everything set up including `live reload`, `postgresql`.

To develop using docker, make sure you have docker installed then run the following command:

```sh
docker compose up
```

To perform some command within the docker environment you can mount into the docker bash with the following command

First get the name of the docker image with the following command

```sh
docker ps -l

# Output should look something like this
# CONTAINER ID   IMAGE                       COMMAND              CREATED         STATUS                     PORTS     NAMES
# 17f0071b0f17   go-api-boilerplate-go_api   "air -c .air.toml"   6 minutes ago   Exited (0) 4 minutes ago             go-api-boilerplate-go_api-1
```

To mount into the docker bash run the following command

```sh
docker exec -it go-api-boilerplate-go_api-1 bash
```

After mount into docker bash, you can perform some action such as migration.

```sh
cd /app
go run ./cmd/migrate/main.go
```

## Local environment

Develop local with hot reload

```sh
air
```

Develop local without hot reload

```sh
go run ./cmd/api/main.go
```

Migrate database

```sh
go run ./cmd/migrate/main.go
```

# Testing

## Unit test

To run all unit tests

```sh
go test ./...
```

To test rate limit

```sh
go test ./internal/rate_limiter
```

To test mail

```sh
go test ./internal/mailer
```

## Script test

To test rate limit by performing http request

```sh
./scripts/test_rate_limit.sh
```
