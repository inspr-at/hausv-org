# WEG Portal

Private multi-tenant house portal for WEG communication, access control, and the
Janischhofweg 22 parking usage area.

## Local Development

```fish
go test ./...
go run .
```

Local-only environment belongs in `.env.local`; it is intentionally ignored.

## Production

csb1 deployment notes live in `docs/csb1-deploy.md`. Secrets stay in agenix and
must not be committed.

## Security Notes

External service-provider access and attachment privacy are documented in
`docs/service-provider-privacy.md`.
