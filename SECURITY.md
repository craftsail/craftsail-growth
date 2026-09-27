# Security policy

## Reporting a vulnerability

Please do not open a public issue. Report it privately through
[GitHub security advisories](https://github.com/craftsail/craftsail-growth/security/advisories/new)
or by email to askuygo@foxmail.com.

Include the version or commit, how to reproduce, and the impact you see. We aim
to reply within 7 days. Once a fix is released, we will credit you
unless you prefer otherwise.

## Supported versions

Only the latest release and the `main` branch receive security fixes.

## Deployment notes

- Set `CRAFTSAIL_GROWTH_PASSWORD` before the first start. Otherwise the
  default `admin` account can do nothing until its public password is
  changed, but anyone who reaches the dashboard first can change it.
- Serve the dashboard over HTTPS; the session cookie and API token must not
  travel over plain HTTP.
- `config/default.toml` holds API keys. Keep it readable only by the service
  user (the app creates it with mode 0600).
