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

HAUSV is a communication, administration and energy management portal. Annual-statement calculations are working drafts based on the period’s configured cost types, allocation keys, unit bases, confirmed receipts and recorded prepayments. Each successful run saves all unit balances and its input snapshot; missing keys, receipts, original files, prepayments or required measurements block the entire run. Earlier runs remain unchanged when inputs are corrected. Each run keeps only the checked consumption vector and the two boundary facts per unit; interior readings are checked for resets in one streaming scan and are not copied into every run. These calculations do not constitute a legal assessment under Austrian WEG/MRG.

The management page is `/app/settings/annual-statement`; `POST /app/settings/annual-statement/runs` calculates the selected `year` using stored data only. Every allocatable cost type requires a confirmed receipt. An explicit zero prepayment is accepted, while a missing row is not. Nutzwert requires a complete total of 1,000,000 ppm; cents are distributed per cost type by largest remainder, with unit-ID order breaking ties. SQLite and PostgreSQL persist each run with a calculation version and input hash.

Heating and hot-water evidence comes from confirmed consumer-energy mappings of heat pumps and hot-water appliances to a home’s unit. The connector and direct Home Assistant sampler preserve cumulative counters with their original timestamps, converting Wh/kWh/MWh exactly to micro-kWh. Calculations require exact start and end boundaries of the period in Europe/Vienna, one unambiguous source per unit and no counter reset; missing boundaries are never estimated. These are measured appliance-energy shares, not an inferred statutory heating allocation rule.

HAUSV does not replace property accounting software or issue annual-statement PDFs, bookkeeping records, dunning notices or payment orders. Structured data can be exchanged with existing systems such as BMD.

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

### Fixture development loop

For click-through development with a fresh, deterministic local fixture (and
automatic rebuilds for `.go` and `.templ` files), run:

```sh
scripts/dev.sh
```

It opens `http://localhost:8098` with local-development login enabled and
removes its temporary data when stopped with Ctrl-C. Use another port with
`HV_DEV_PORT=8100 scripts/dev.sh`. If [`templ`](https://templ.guide/) is
installed, its generator runs in watch mode too; without it the script keeps
running and reports that generation is unavailable.

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
