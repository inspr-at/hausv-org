# hausv.org

Multi-tenant platform with two product surfaces on one secure core; the core is
open source under the GNU AGPL-3.0 (see [LICENSE](LICENSE)), while production
deployments and tenant data remain private:

- **HAUSV Gemeinschaft** supports communication, transparency, and self-service
  for WEGs and other multi-unit buildings.
- **HAUSV Zuhause** helps private apartments and houses understand their current
  state and identify the next sensible task, including read-only energy insights.

Both surfaces share houses, people, roles, assets, documents, tasks,
measurements, recommendations, and audit. The Janischhofweg 22 tenant is the
first production deployment and includes both community workflows and the first
private-home energy pilot.

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

Repeatable role-aware browser QA and the privacy-safe structured-log check are
documented in `docs/playwright-main-flow-qa.md` and
`docs/structured-log-qa.md`.
