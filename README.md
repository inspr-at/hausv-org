# HAUSV

HAUSV is an open source portal for property communication, administration, and energy management. It supports self-managed homes, resident communities, and professional property managers in one multi-tenant application.

## Products

| Product | Operation | Price | Support |
| --- | --- | --- | --- |
| **HAUSV Free** | Self-hosted | Free under the GNU AGPL-3.0 | [GitHub issues](https://github.com/inspr-at/hausv-org/issues) and [pull requests](https://github.com/inspr-at/hausv-org/pulls) |
| **HAUSV Home** | Hosted | 12 months free, then 12 EUR per year | Email |
| **HAUSV Professional** | Hosted or self-hosted | First 25 units free, then 3 EUR per additional unit and month | Email and phone |

## Features

- Central overview of tasks, appointments, announcements, open issues, and energy status
- Announcements and targeted resident communication
- Appointments with personal calendar feeds
- Issue and damage workflows with files, photos, priorities, assignments, and resolution confirmation
- Protected document storage with visibility rules, previews, downloads, and version history
- Digital apartment handovers with rooms, condition, meter readings, keys, parties, and attachments
- Voting with traceable results and records
- Role-based access for managers, boards, residents, owners, and service providers
- Live energy flows for solar, grid, batteries, homes, and individual consumers through direct Home Assistant integration or the outbound-only local connector
- Parking, charging, CAMT, ebInterface, and structured data exchange with specialist systems

## Scope

HAUSV is a communication and energy management portal. It does not replace property accounting software and does not produce annual statements, bookkeeping records, dunning notices, or payment orders. Structured data can be exchanged with existing systems such as BMD.

## Development

Requirements:

- Go 1.26.6
- A local `.env.local` file for development settings

Run the application:

```sh
go test ./...
go run ./cmd/hausv-org
```

`.env.local` is ignored by Git. Do not commit credentials or production configuration.

## Tenant URLs

One installation serves multiple portals below one domain. Each tenant has a
stable slug and is available at `https://hausv.org/<slug>`, for example
`https://hausv.org/demo`. Adding a tenant does not require a DNS change.

Tenant metadata is supplied through `WEG_TENANTS_JSON`. Hosting paths, operator
details, Home Assistant credentials and other deployment-specific values stay
outside the repository. See the deployment documentation for the required
runtime settings.

## Documentation

- [Deployment](docs/production-deploy.md)
- [Browser QA](docs/playwright-main-flow-qa.md)
- [Service provider privacy](docs/service-provider-privacy.md)
- [Structured logging QA](docs/structured-log-qa.md)
- [Supply chain pins](docs/supply-chain-pins.md)

## License

The open source core is licensed under the [GNU AGPL-3.0](LICENSE).
