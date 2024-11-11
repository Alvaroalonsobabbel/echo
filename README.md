# echo

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/Alvaroalonsobabbel/echo) ![Test](https://github.com/Alvaroalonsobabbel/echo/actions/workflows/test.yaml/badge.svg)

Echo serves ephemeral/mock endpoints created with parameters specified by clients and it's based on [these requirements](echo.md).

## Technical information

This application was built using Go, and the endpoints use [JSON:API v1.0](https://jsonapi.org/) as a format.

## Run locally

1. [Install Go](https://go.dev/doc/install)
2. (optional) Run the tests with `make test`
3. (optional) Run the linter with `make lint` - you have to have [golangci-lint](https://golangci-lint.run/welcome/install/) installed.
4. Start the server with `make run`

Use cURL or Postman to send HTTP requests to the server at: `http://localhost:3000`

The Server works using the exact API documentation specified in the [requirements' examples](echo.md#examples).

To alleviate the burden of the tester, this implementation comes with an in-memory DB and four seeded endpoints.

## Quick cURL commands to test the server

View endpoints:

```bash
curl -L -X GET 'http://127.0.0.1:3000/endpoints'
```

Query an existing endpoint:

```bash
curl -L -X GET 'http://127.0.0.1:3000/revert_entropy'
```

Submit an endpoint:

```bash
curl -L -X POST 'http://127.0.0.1:3000/endpoints' \
-H 'Content-Type: application/vnd.api+json' \
-d '{
    "data": {
        "type": "endpoints",
        "attributes": {
            "verb": "GET",
            "path": "/say_hi",
            "response": {
              "code": 200,
              "headers": {"Content-Type":"text/plain"},
              "body": "hi!"
            }
        }
    }
}'
```

Now you can run View endpoints again to check the endpoint is there.

Use the endpoint:

```bash
curl -L -X GET 'http://127.0.0.1:3000/say_hi'
```
