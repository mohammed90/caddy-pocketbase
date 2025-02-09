# Caddy PocketBase Module

A Caddy module that integrates [PocketBase](https://pocketbase.io/)([Repository](https://github.com/pocketbase/pocketbase)) as a Caddy application, allowing you to run PocketBase embedded in your Caddy server.

## Features

- Run PocketBase as a native Caddy module
- Admin API endpoints for managing superusers
- Configurable data directory and origins
- Automatic port allocation if none specified
- Full integration with Caddy's configuration and lifecycle

## Build

There are 2 ways to build.

This is the easiest:


```sh
cd cmd && go build .
```

If your into xcaddy:

```sh
xcaddy --with github.com/mohammed90/caddy-pocketbase
```


## Configuration

Example Caddyfile configuration:

```caddyfile
{
    order pocketbase before file_server
}
example.com {
    pocketbase
}
```

## Key Components

- **PocketBase Integration**: Runs PocketBase within Caddy.
- **Admin API**: Provides endpoints for superuser management.
- **Configuration**: Allows customization of data directory and origins.


## Usage

This module enables you to run PocketBase as part of your Caddy server, simplifying deployment and management of both services. The configuration options allow for easy customization to fit various deployment scenarios.

## Admin API Endpoints
The module provides admin API endpoints under `/pocketbase/`:

- `POST /pocketbase/superuser` - Create a new superuser
- `PUT /pocketbase/superuser` - Upsert a superuser
- `PATCH /pocketbase/superuser` - Update superuser password
- `DELETE /pocketbase/superuser` - Delete a superuser
- `POST /pocketbase/superuser/{email}/otp` - Generate OTP for superuser

All the above endpoints require a JSON payload, except for the OTP endpoint. The
JSON payload for the superuser endpoints is as follows:

```json
{
		"email_address": "...",
		"password": "..."
}
```

The `DELETE` endpoint does not expect the `password` field.
