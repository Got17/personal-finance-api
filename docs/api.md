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
Requires a valid Bearer token. Accepts a JSON payload with one of the supported
currency codes (`base_currency`; see [`GET /v1/currencies`](#get-v1currencies)).
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

## Currencies

### `GET /v1/currencies`

Returns the fixed, ordered list of currencies accepted by this system. Public;
does not require a Bearer token.

Successful responses return `200 OK` with JSON matching this shape:

```json
{
  "success": true,
  "data": [
    { "code": "LAK", "name": "Lao Kip", "symbol": "₭", "decimal_digits": 0 },
    { "code": "THB", "name": "Thai Baht", "symbol": "฿", "decimal_digits": 2 },
    { "code": "USD", "name": "US Dollar", "symbol": "$", "decimal_digits": 2 },
    { "code": "CNY", "name": "Chinese Yuan", "symbol": "¥", "decimal_digits": 2 },
    { "code": "EUR", "name": "Euro", "symbol": "€", "decimal_digits": 2 }
  ],
  "message": ""
}
```

Every `currency` and `base_currency` field elsewhere in this API (Accounts,
Financial Records, user preferences) accepts only one of these five codes.

## Accounts

### `POST /v1/accounts`

Creates a new financial account for the currently authenticated user.
Requires a valid Bearer token in the `Authorization` header.
Accepts `name`, `type` (`checking`, `savings`, `investment`, `cash`, `other`), `currency` (one of the supported codes from [`GET /v1/currencies`](#get-v1currencies)), optional `description`, and optional `is_active` (defaults to `true`).
Invalid input or unsupported account type/currency returns `422 Unprocessable Entity`.

```json
{
  "name": "Main Checking",
  "type": "checking",
  "currency": "USD",
  "description": "Primary daily checking account"
}
```

Successful responses return `201 Created` with the created account:

```json
{
  "success": true,
  "data": {
    "id": "c7a8e999-4c0b-4ef8-bb6d-6bb9bd380a22",
    "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    "name": "Main Checking",
    "type": "checking",
    "currency": "USD",
    "description": "Primary daily checking account",
    "is_active": true,
    "created_at": "2026-09-06T12:00:00Z",
    "updated_at": "2026-09-06T12:00:00Z"
  },
  "message": ""
}
```

### `GET /v1/accounts`

Retrieves all accounts belonging strictly to the currently authenticated user.
Requires a valid Bearer token. Callers can only list their own accounts.

Successful responses return `200 OK` with an array of accounts:

```json
{
  "success": true,
  "data": [
    {
      "id": "c7a8e999-4c0b-4ef8-bb6d-6bb9bd380a22",
      "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
      "name": "Main Checking",
      "type": "checking",
      "currency": "USD",
      "description": "Primary daily checking account",
      "is_active": true,
      "created_at": "2026-09-06T12:00:00Z",
      "updated_at": "2026-09-06T12:00:00Z"
    }
  ],
  "message": ""
}
```

### `PUT /v1/accounts/:id` & `PATCH /v1/accounts/:id`

Updates mutable details of an existing account belonging strictly to the currently authenticated user.
Requires a valid Bearer token. Callers can only update their own accounts (cross-user updates return `403 Forbidden`).
Accepts optional fields `name`, `type`, `currency`, `description`, and `is_active`.
Invalid input or unsupported account type/currency returns `422 Unprocessable Entity`.
If the account ID does not exist, returns `404 Not Found`.

```json
{
  "name": "Updated Checking",
  "description": "Updated description"
}
```

Successful responses return `200 OK` with the updated account:

```json
{
  "success": true,
  "data": {
    "id": "c7a8e999-4c0b-4ef8-bb6d-6bb9bd380a22",
    "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    "name": "Updated Checking",
    "type": "checking",
    "currency": "USD",
    "description": "Updated description",
    "is_active": true,
    "created_at": "2026-09-06T12:00:00Z",
    "updated_at": "2026-09-06T12:00:00Z"
  },
  "message": ""
}
```

### `DELETE /v1/accounts/:id`

Deactivates an account belonging strictly to the currently authenticated user (`is_active` set to `false`) without erasing the record from the database.
Requires a valid Bearer token. Callers can only deactivate their own accounts (cross-user deactivation returns `403 Forbidden`).
If the account ID does not exist, returns `404 Not Found`.

Successful responses return `200 OK` with the deactivated account:

```json
{
  "success": true,
  "data": {
    "id": "c7a8e999-4c0b-4ef8-bb6d-6bb9bd380a22",
    "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    "name": "Updated Checking",
    "type": "checking",
    "currency": "USD",
    "description": "Updated description",
    "is_active": false,
    "created_at": "2026-09-06T12:00:00Z",
    "updated_at": "2026-09-06T12:00:00Z"
  },
  "message": ""
}
```

## Categories

### `POST /v1/categories`

Creates a new category for the currently authenticated user.
Requires a valid Bearer token in the `Authorization` header.
Accepts `name`, `type` (`income` or `expense`), and optional `is_active` (defaults to `true`).
Invalid input or unsupported category type returns `422 Unprocessable Entity`.

```json
{
  "name": "Salary",
  "type": "income"
}
```

Successful responses return `201 Created` with the created category:

```json
{
  "success": true,
  "data": {
    "id": "e8b9f000-5d1c-4fe9-cc7e-7cc0ce491b33",
    "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    "name": "Salary",
    "type": "income",
    "is_active": true,
    "created_at": "2026-09-08T12:00:00Z",
    "updated_at": "2026-09-08T12:00:00Z"
  },
  "message": ""
}
```

### `GET /v1/categories`

Retrieves all categories belonging strictly to the currently authenticated user.
Requires a valid Bearer token. Callers can only list their own categories.

Successful responses return `200 OK` with an array of categories:

```json
{
  "success": true,
  "data": [
    {
      "id": "e8b9f000-5d1c-4fe9-cc7e-7cc0ce491b33",
      "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
      "name": "Salary",
      "type": "income",
      "is_active": true,
      "created_at": "2026-09-08T12:00:00Z",
      "updated_at": "2026-09-08T12:00:00Z"
    }
  ],
  "message": ""
}
```

### `PUT /v1/categories/:id` & `PATCH /v1/categories/:id`

Updates mutable details of an existing category belonging strictly to the currently authenticated user.
Requires a valid Bearer token. Callers can only update their own categories (cross-user updates return `403 Forbidden`).
Accepts optional fields `name`, `type` (`income` or `expense`), and `is_active`.
Invalid input or unsupported category type returns `422 Unprocessable Entity`.
If the category ID does not exist, returns `404 Not Found`.

```json
{
  "name": "Freelance Work",
  "type": "income"
}
```

Successful responses return `200 OK` with the updated category:

```json
{
  "success": true,
  "data": {
    "id": "e8b9f000-5d1c-4fe9-cc7e-7cc0ce491b33",
    "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    "name": "Freelance Work",
    "type": "income",
    "is_active": true,
    "created_at": "2026-09-08T12:00:00Z",
    "updated_at": "2026-09-08T12:00:00Z"
  },
  "message": ""
}
```

### `DELETE /v1/categories/:id`

Deactivates a category belonging strictly to the currently authenticated user (`is_active` set to `false`) without erasing the record from the database.
Requires a valid Bearer token. Callers can only deactivate their own categories (cross-user deactivation returns `403 Forbidden`).
If the category ID does not exist, returns `404 Not Found`.

Successful responses return `200 OK` with the deactivated category:

```json
{
  "success": true,
  "data": {
    "id": "e8b9f000-5d1c-4fe9-cc7e-7cc0ce491b33",
    "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    "name": "Freelance Work",
    "type": "income",
    "is_active": false,
    "created_at": "2026-09-08T12:00:00Z",
    "updated_at": "2026-09-08T12:00:00Z"
  },
  "message": ""
}
```

## Financial Records

### `POST /v1/financial-records`

Creates an active Income or Expense Financial Record for the authenticated user.
The request requires `kind` (`income` or `expense`), `account_id`, `category_id`,
positive integer `amount_minor`, `currency`, and an RFC 3339 `date`; `note` is
optional. `amount_minor` is never a floating-point amount. The Account and
Category must be active, owned by the caller, and their Category type must
match `kind`. The request currency must equal the Account currency.

```json
{
  "kind": "expense",
  "account_id": "c7a8e999-4c0b-4ef8-bb6d-6bb9bd380a22",
  "category_id": "e8b9f000-5d1c-4fe9-cc7e-7cc0ce491b33",
  "amount_minor": 4250,
  "currency": "USD",
  "date": "2026-09-10T00:00:00Z",
  "note": "Groceries"
}
```

Returns `201 Created`. Malformed JSON returns `400`; unauthenticated calls
return `401`; foreign references return `403`; missing references return `404`;
and invalid amounts, currencies, dates, inactive references, or mismatched
Category kinds return `422`.

### `GET /v1/financial-records`

Lists only the caller's records, newest first by record date. It defaults to
active records. Optional query filters are `start_date`, `end_date` (both
`YYYY-MM-DD`), `kind` (`income`, `expense`, or `transfer`), `account_id`, `category_id`, and
`include_archived=true`. When filtering by `account_id`, records where the account is
either the source or destination are returned. Transfers have `category_id: null` and
include destination account and currency details. Invalid query values return `400`;
an invalid kind or date range returns `422`.

### `GET /v1/financial-records/:id`

Retrieves one Financial Record owned by the authenticated user. Records owned
by another user return `403`; absent records return `404`.

### `PUT /v1/financial-records/:id` & `PATCH /v1/financial-records/:id`

Corrects an active Financial Record owned by the authenticated user. Every field
is optional, but the resulting record must still use active, owned Account and
Category references whose Category type matches `kind`; its currency must match
the Account currency, `amount_minor` must be positive, and `date` must be valid.
Archived records are terminal and return `422 Unprocessable Entity` when edited.
Cross-user requests return `403 Forbidden`.

```json
{
  "amount_minor": 4250,
  "note": "Corrected grocery total"
}
```

### `DELETE /v1/financial-records/:id`

Archives an owned Financial Record by setting `is_active` to `false`; it never
hard-deletes financial history. Archived records remain individually retrievable
and are returned by history only with `include_archived=true`. Cross-user archive
requests return `403 Forbidden`; attempting to archive an already archived record
returns `422 Unprocessable Entity`.

## Transfers

### `POST /v1/transfers`

Executes an atomic transfer between two distinct, active, owned Accounts of the authenticated user.
Transfers can be same-currency or cross-currency:
- **Same-currency**: `source_currency` equals `destination_currency`. `source_amount_minor` must equal `destination_amount_minor` (or `destination_amount_minor` can be omitted, defaulting to `source_amount_minor`). No FX quote is required or stored.
- **Cross-currency**: `source_currency` does not equal `destination_currency`. Requires either an explicit `rate` override or an automatic rate lookup from the FX quote provider for the transfer date. Destination amount is validated against currency precision: $\text{round}(\text{amount}_{src} \times \text{rate} \times 10^{\text{digits}_{dest} - \text{digits}_{src}})$. An immutable `HistoricalFXQuote` is captured and linked with provenance (`provider` or `manual_override`).
- **Optional Fee**: An optional `fee` payload charges an active fee account in its own currency, categorized under an active expense Category. The fee is persisted atomically as a linked `expense` financial record (`linked_transfer_id`).
- **Available Balance Validation**: Outbound transfers validate that the source account holds sufficient funds (`available_balance >= source_amount_minor`, including fee amount if charged to source). If the fee is charged to a separate account, that account must also hold sufficient balance. Returns `422 Unprocessable Entity` with descriptive top-level message (`"Source account has insufficient balance to complete the transfer"` or `"Fee account has insufficient balance to pay the transfer fee"`) along with targeted field errors (`{"source_account_id": "insufficient account balance"}` or `{"fee.account_id": "insufficient account balance"}`).
- Transfers have no Category and do not contribute to cash-flow / Category spending totals.

```json
{
  "source_account_id": "c7a8e999-4c0b-4ef8-bb6d-6bb9bd380a22",
  "destination_account_id": "d8b9f000-5d1c-4fe9-cc7e-7cc0ce491b44",
  "source_amount_minor": 10000,
  "destination_amount_minor": 9200,
  "date": "2026-09-15T00:00:00Z",
  "note": "Cross-currency transfer USD -> EUR",
  "fee": {
    "account_id": "c7a8e999-4c0b-4ef8-bb6d-6bb9bd380a22",
    "category_id": "e8b9f000-5d1c-4fe9-cc7e-7cc0ce491b33",
    "amount_minor": 200,
    "note": "Wire fee"
  }
}
```

Returns `201 Created` with the complete Transfer representation, including linked FX quote and fee details if applicable.

### `GET /v1/transfers`

Lists all transfers owned by the authenticated user, sorted by date descending.

### `GET /v1/transfers/:id`

Retrieves a single transfer by ID owned by the authenticated user. Foreign transfers return `403 Forbidden`; non-existent transfers return `404 Not Found`.

## Foreign Exchange Quotes

### `GET /v1/fx-quotes`

Retrieves the reference foreign exchange rate between two supported currencies for a given date.

Query parameters:
- `from`: Source currency code (e.g. `USD`). Required.
- `to`: Destination currency code (e.g. `EUR`). Required.
- `date`: Date in RFC 3339 format or `YYYY-MM-DD`. Optional (defaults to current UTC date).

```json
{
  "success": true,
  "data": {
    "from_currency": "USD",
    "to_currency": "EUR",
    "rate": 0.92,
    "date": "2026-09-15T00:00:00Z"
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
