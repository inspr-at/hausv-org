# Art. 28 and TOM approval template

Status: Draft. Complete and approve this document before enabling service
provider access for a production tenant.

This template supplements the unchanged
[EU standard contractual clauses 2021/915](https://eur-lex.europa.eu/eli/dec_impl/2021/915/oj?locale=en).
It contains no operator, customer or infrastructure-specific data by design.

## Parties

| Role | Legal name | Address | Authorized signatory |
|---|---|---|---|
| Controller | To be completed | To be completed | To be completed |
| Processor | To be completed | To be completed | To be completed |

Record the approval date, contract version and evidence outside the public
repository. Keep `SERVICE_PROVIDER_ACCESS_ENABLED=false` until approval is
complete.

## Processing

| Item | Agreement |
|---|---|
| Subject | Technical operation of a HAUSV tenant portal |
| Duration | From documented approval until termination and agreed deletion or return |
| Purpose | Building communication, documents, appointments, issue handling, votes, access control, handovers, parking and optional read-only energy functions |
| Data subjects | Owners, residents, property managers, advisory boards, invited service providers and technical administrators |
| Data | Identity and contact data, tenant and unit membership, roles, audit events, portal content, attachments, parking values and explicitly configured energy mappings |

## Controller instructions

The processor acts only on documented instructions. Access is tenant-scoped,
role-scoped and revocable. Service providers may see only open issues assigned
to them. Product analytics, advertising and model training are excluded unless
separately instructed with a valid legal basis.

## Technical and organizational measures

- encrypted transport and secure cookies;
- tenant and role isolation with server-side authorization;
- least-privilege administrative and service-provider access;
- secrets outside the repository and application database;
- encrypted backups with documented retention and restore tests;
- structured logs without bearer tokens or unnecessary personal identifiers;
- audit records for security-relevant changes;
- documented incident, deletion and access-revocation procedures;
- dependency and container checks in CI;
- release verification and recoverable deployment.

## Subprocessors and transfers

Complete this table from the actual deployment configuration. Do not copy
vendors from another operator.

| Provider | Purpose | Region and transfer basis | DPA and TOM evidence |
|---|---|---|---|
| Hosting provider | Application and identity service | To be completed | To be completed |
| Web protection provider | DNS, reverse proxy and abuse protection | To be completed | To be completed |
| Mail provider | Transactional email | To be completed | To be completed |
| Backup provider | Encrypted off-site backups | To be completed | To be completed |

Any new recipient, processing region or purpose closes the approval gate until
the assessment and affected notices are updated.

## Retention and deletion

Record the production retention schedule for accounts, content, audit events,
attachments, energy imports and backups. On termination, return or delete data
according to the controller's documented instruction, subject only to binding
legal retention duties.

## Approval checklist

- [ ] Parties and authorized signatories confirmed
- [ ] Purpose, data subjects and data categories confirmed
- [ ] Subprocessors, regions and transfer mechanisms verified
- [ ] DPA and TOM evidence archived outside the repository
- [ ] Retention and deletion schedule approved
- [ ] Incident contacts and response process tested
- [ ] Tenant, role and service-provider access tested
- [ ] Backup restore tested
- [ ] Public privacy notice matches the actual deployment
- [ ] Approval evidence recorded in PPM
