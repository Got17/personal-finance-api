# Personal Finance API

## Sign in

### `POST /v1/auth/sign-in`

Authenticates a provisioned user with an email address and password, returning
a signed JWT bearer token. The endpoint never returns a password or password
hash. Invalid email addresses and passwords receive the same `401` response,
so the endpoint does not reveal whether an account exists.

```json
{
  "email": "owner@example.com",
  "password": "your-password"
}
```

Successful responses return `200 OK` with an `access_token` and a
`token_type` of `Bearer`. Send the token on protected requests as
`Authorization: Bearer <access_token>`.

## Health

### `GET /v1/health`

Returns the availability of the Personal Finance API. This endpoint is public
so deployment tooling can check the service before authentication is configured.

Successful responses return `200 OK` with JSON matching this shape:

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "app": "personal-finance-api"
  },
  "message": ""
}
```

`data.app` is the configured application name. The machine-readable source of
truth is [`api/openapi.yaml`](../api/openapi.yaml).
