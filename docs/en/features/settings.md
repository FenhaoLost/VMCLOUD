# System Settings

The System Settings page centralizes panel-level configuration, with the following groups.

## SSL Certificates

- Supports four modes: disabled, Let's Encrypt, self-signed, and uploaded certificate.
- After configuring the target domain and email, Let's Encrypt certificates can be issued/renewed automatically.
- After saving, restart the service immediately to apply, or restart the `systemd` service manually later.

## Panel Access Sources

- Enables the panel access allowlist, allowing only specified IPs/networks to reach the web panel.
- Configures trusted reverse proxy addresses to avoid false blocks behind a proxy.
- Equivalent to the `eyvescloud access-policy` CLI; takes effect immediately after saving.

## Notifications

- Security alerts can be pushed via Webhook.
- SMTP email push is supported; server, port, account, and recipients are configurable.

## Task Queue

- Configure the task queue concurrency to control how many reinstall/create/image tasks run at the same time.
- Takes effect immediately after saving.

## Account

- Change the administrator username and password.
- The current password is required for confirmation.

## Miscellaneous

- WebSSH/WebVNC Origin allowlist.
- Language setting (Simplified Chinese / English).

## Related Endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET/PUT | `/api/ssl` | SSL settings |
| GET/PUT | `/api/access-policy` | Panel access source policy |
| GET/PUT | `/api/notifications` | Notification push settings |
| GET/PUT | `/api/task-queue/settings` | Task queue concurrency |
| POST | `/api/change-password` | Change password |
| POST | `/api/change-username` | Change username |
| GET/PUT | `/api/webssh-origins` | WebSSH/WebVNC Origin allowlist |
