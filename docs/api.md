# Personal Finance API

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
