# Personal Finance API

## Authentication & Identity

### `POST /v1/auth/login`

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

### `POST /v1/auth/signup`

Registers a new user account with an email address and password and automatically
provisions a primary private workspace. Returns `201 Created` with a session
JWT token. Duplicate email registration attempts receive a `409 Conflict` response.

```json
{
  "email": "newuser@example.com",
  "password": "securepassword123",
  "workspace_name": "Personal Workspace"
}
```

### `GET /v1/users/me`

Retrieves the identity and profile of the currently authenticated user.
Requires a valid Bearer token in the `Authorization` header (`Authorization: Bearer <access_token>`).
Unauthenticated requests or invalid/expired tokens are rejected with a `401 Unauthorized` response.
User identity is derived strictly from the validated token session, preventing callers from accessing or selecting another user's identity or workspace.

Successful responses return `200 OK` with JSON matching this shape:

```json
{
  "success": true,
  "data": {
    "id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    "email": "owner@example.com",
    "base_currency": "USD",
    "created_at": "2026-09-06T12:00:00Z",
    "updated_at": "2026-09-06T12:00:00Z"
  },
  "message": ""
}
```

### `GET /v1/users/me/preferences`

Retrieves the base currency preference of the currently authenticated user.
Requires a valid Bearer token. Returns `200 OK`:

```json
{
  "success": true,
  "data": {
    "base_currency": "USD"
  },
  "message": ""
}
```

### `PUT /v1/users/me/preferences` & `PATCH /v1/users/me/preferences`

Updates the base currency preference for the currently authenticated user.
Requires a valid Bearer token. Accepts a JSON payload with a valid 3-letter ISO 4217 currency code (`base_currency`).
Invalid or unsupported currency codes receive a `422 Unprocessable Entity` response.

```json
{
  "base_currency": "EUR"
}
```

Successful responses return `200 OK`:

```json
{
  "success": true,
  "data": {
    "base_currency": "EUR"
  },
  "message": ""
}
```

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
