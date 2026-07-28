# hausv.org

Private multi-tenant portal for communication, transparency, and self-service in
WEGs and other multi-unit buildings. The Janischhofweg 22 tenant is the first
production deployment and includes the parking-usage area.

hausv.org deliberately complements existing accounting and property-management
systems. It does not implement bookkeeping, tax logic, dunning, or payment
orders.

## Local Development

```fish
go test ./...
go run ./cmd/hausv-org
```

Local-only environment belongs in `.env.local`; it is intentionally ignored.

The repeatable Playwright role check for the main portal flows is documented in
[`docs/playwright-main-flow-qa.md`](docs/playwright-main-flow-qa.md).

## Production

csb1 deployment notes live in `docs/csb1-deploy.md`. Secrets stay in agenix and
must not be committed.

## Security Notes

External service-provider access and attachment privacy are documented in
`docs/service-provider-privacy.md`.
