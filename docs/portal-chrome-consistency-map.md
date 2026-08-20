# Portal left-navigation chrome map

Baseline: HAUSV 0.98.11, tenant `jhw22`. This is an implementation map and a scoped component-unification recommendation. It does not propose feature, route, permission, or workflow changes.

The source navigation uses canonical `/app...` paths. `tenantPaths` and `prefixTenantHTMLPaths` in `internal/server/server.go` expose those paths as `/jhw22/app...` on the live tenant. `PortalNavigation` in `internal/web/portal.templ` is the single source for desktop and mobile menu order, labels, routes, gates, active state, and badges.

## Current diagnosis

The product already shares `PortalDocument`, `PortalSidebar`, `PortalNavigation`, `PortalMobileHeader`, and `portalMapForTenant`, but it does not share section-level chrome. The 0.98.11 landings render four general header families—home hero, the local events hero, icon crumb, and eyebrow—plus settings, energy, and help variants.

The “Termine” defect is evidence of that split, not the whole problem:

- `EventsContent` in `internal/web/events.templ` copies `.home-hero`, `.home-hero-image`, `.home-hero::after`, and `.home-hero-copy` instead of calling a shared partial.
- `.events-main` retains `padding: var(--space-6)` above a normal-flow, `position: relative` hero, which creates the rejected 24px top gap.
- `EventCreateButton` uses filled `.button.primary`, while the approved “Hausüberblick” hero action is outline `.button`.
- Contacts, documents, and handovers copy `page-top`/`crumb`/heading rules; announcements, issues, ballots, parking, audit, and settings copy `page-head`/`eyebrow`/action rules. Several route styles also redeclare shell and mobile-header CSS.

The clean eyebrow hierarchy on “Abstimmungen” and the approved photo treatment on “Hausüberblick” should become two configurations of one kit, not two more page-owned families.

## One shared chrome component kit

All components and chrome CSS should live in `internal/web/portal.templ`. A page may omit the optional hero, but it must compose the same kit and must not define route-local `.home-hero`, `.page-head`, or `.crumb` rules.

| Kit part | Responsibility |
| --- | --- |
| `PortalDocument` | Existing document, assets, tokens, and shared style composition. Route styles remain for feature content only. |
| `PortalShell` | New shared skip link + `PortalMobileHeader` + `.shell` composition. It accepts the portal context, mobile label, and main-content slot. |
| `PortalSidebar` | Existing desktop identity/navigation component: flush map, one small address, quiet portal switcher, `PortalNavigation`, and account footer. |
| `PortalSectionLanding` | New normal-flow main container. It owns top-edge behavior, content width, horizontal gutters, vertical rhythm, and the slots below. |
| `PortalSectionHero` | Optional shared photo band. It owns the existing “Hausüberblick” image crop, gradient, full-width bleed, and bottom hairline; it accepts header and action slots. No page may clone its markup or CSS. |
| `PortalSectionHeader` | Shared header whether rendered inside `PortalSectionHero` or directly in `PortalSectionLanding`. It owns alignment, responsive stacking, and title/lede/action geometry. |
| `PortalSectionIdentity` | Shared quiet identity line. Primary landings use the eyebrow form; an icon or back-link is data passed to this partial, not a page-specific crumb implementation. |
| `title` slot | The single page `h1`; existing visitor-facing titles remain unchanged. |
| `lede` slot | Optional explanatory copy with one shared width, type scale, and spacing rule. |
| `action` slot | Optional header action area. Any action placed here uses outline `.button`; filled `.button.primary` is reserved for in-content next steps and form/dialog submissions. |
| `status` slot | Optional full-width status/control row between the outer shell and section body. `EnergyModeStrip` plugs into this slot unchanged. |
| `context` slot | Optional compact context row for a board back-link, scoped status, or secondary utility. It uses shared kit styles rather than a route-local crumb. |
| `content` slot | Existing feature body. Cards, tables, forms, filters, workflows, and empty states stay in their current template files. |

`PortalHeroGreeting` remains a small home-specific content partial, but it renders inside `PortalSectionHero`. `SettingsFrame` becomes a thin composer of `PortalShell` and `PortalSectionLanding`; it no longer owns a separate header system. `EnergyModeStrip` remains its existing feature partial and is only hosted by the shared `status` slot.

## Landing decisions

Hero presence is decided from the landing’s job. The photo is useful for ambient, house-wide orientation and editorial/calendar context; it is omitted when it would delay a task, status, search, or administrative workflow. A no-hero page still uses the same `PortalSectionHeader`, identity, title, lede, action, and spacing components.

| Label | Route | Templ | Hero? | Action slot | Extra slots | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| “Hausüberblick” | `/app` | `internal/web/portal.templ`: `PortalPage` → `PortalContent` → `PortalDense`/`PortalCalm`; `PortalMobile` | **Yes** — it is the approved house-orientation baseline and the photo carries tenant identity before dashboard content. | Outline “Anliegen melden” when permitted. | Existing greeting, metrics, dense/calm content, and mobile composition. | Keep `PortalHeroGreeting` output and geometry visually unchanged; `portal` handler is in `internal/server/server.go`. |
| “Mein Zuhause” | `/app/energie` | `internal/web/energy.templ`: `EnergyPage` → `EnergyModeStrip`, `EnergyHeading` | **No** — live mode, tariff peak, household identity, and safety controls must remain visible without a decorative band pushing them down. | Outline “Zuhause bearbeiten”; mode controls stay in the status slot. | `status`: existing `EnergyModeStrip`; energy flashes and cockpit body. | Preserve HAUSV-558 wrapping, sticky offsets, and container queries; handler `energyCockpit` is in `internal/server/energy.go`. |
| “Aushang” | `/app/announcements` | `internal/web/announcements.templ`: `AnnouncementsPage` → `AnnouncementsContent` | **Yes** — it is the house-wide editorial landing, so the shared tenant photo provides useful community context before the feed. | Outline “Aushang erstellen” when permitted. | Feed counts and filters remain in content. | Replace the locally dropped-low filled action and route-owned eyebrow CSS; handler in `internal/server/announcements.go`. |
| “Termine” | `/app/events` | `internal/web/events.templ`: `EventsPage` → `EventsContent` | **Yes** — the house calendar is an ambient, house-wide landing where the same tenant photo is useful, provided it is the shared zero-gap hero. | Outline “Termin erstellen” when permitted. | Calendar-feed utility and event counts remain in content. | Delete the local hero clone; the gap and wrapping filled button are symptoms of missing shared composition. Handler in `internal/server/events.go`. |
| “Kontakte” | `/app/kontakte` | `internal/web/contacts.templ`: `ContactsPage` → `ContactsContent` | **No** — this is a lookup task, and the directory and emergency contacts should begin immediately below the shared compact header. | Outline “Kontakt hinzufügen” when permitted. | Optional directory scope/status in context; contact cards in content. | Replace the icon-crumb clone and filled header action; handler in `internal/server/contacts.go`. |
| “Dokumente” | `/app/dokumente` | `internal/web/documents.templ`: `DocumentsPage` → `DocumentsContent` | **No** — search, upload, and retrieval are the primary jobs, so a photo would add scroll before the library. | Outline “Hochladen”; import remains a secondary outline utility. | Search/sort/filter toolbar stays in content. | Replace local crumb, heading, shell, and button chrome; handler in `internal/server/documents.go`. |
| “Anliegen” | Resident `/app/anliegen`; manager `/app/anliegen/board` | `internal/web/issues.templ`: `IssuesPage` → `IssuesContent`; `internal/web/issue_board.templ`: `IssueBoardPage` → `IssueBoardContent` | **No** — reporting, status, and triage are workflow-first and should expose current work immediately. | None on the resident landing; reporting stays in-content filled. Board utilities are outline. | `context`: board back-link and optional calendar utility; issue status/content below. | Manager navigation selects the board route; triage/detail remain drill-down flows. Handlers are in `internal/server/issues.go`. |
| “Abstimmungen” | `/app/abstimmungen` | `internal/web/ballots.templ`: `BallotsPage` | **No** — the current eyebrow hierarchy is the cleanest task-focused reference, and voting state should remain above the fold. | Outline “Abstimmung anlegen” when permitted. | Vote overview/status remains immediately below the header. | Use its hierarchy as the no-hero kit baseline, but remove its route-local `page-head` and button CSS. Handler in `internal/server/votes.go`. |
| “Parkplatznutzung” | `/app/parking` | `internal/web/parking.templ`: `ParkingPage` → `ParkingContent` | **No** — live charging state and monthly cost are operational data that should not be displaced by a photo. | Outline “Mehr” utility; charging and setup actions stay in-content filled. | Live status/switcher and accounting content. | Keep the visitor-facing page title “Parkplatz”; handler in `internal/server/parking.go`. |
| “Übergaben” | `/app/uebergaben` | `internal/web/handovers.templ`: `HandoversPage` → `HandoversContent` | **No** — it is a managed workflow whose counts and next steps matter more than ambient house identity. | Outline “Übergabe anlegen” when permitted. | Metrics/status summary remains in content. | Replace the crumb and `handover-page-head` variation; handler in `internal/server/handover.go`. |
| “Benutzer & Rechte” | `/app/settings/users` | `internal/web/settings.templ`: `UserSettingsPage` → `SettingsFrame` → `UserSettingsContent` | **No** — invitations, access state, and permissions are administrative work and need the compact shared header. | None; invite disclosure and filled submit remain in content. | Settings context identity; user metrics and invite form. | Keep settings form structure, but compose the common kit. Handler in `internal/server/server.go`; renderer in `internal/server/settings_page.go`. |
| “Verlauf” | `/app/audit` | `internal/web/audit.templ`: `AuditPage` | **No** — filtering and reading the evidence stream is the page’s purpose, so the audit summary should remain close to the top. | Outline “Einstellungen” utility when available. | Audit scope/status and filter controls remain in content. | Visitor-facing title may vary by audit scope; handler in `internal/server/server.go`, renderer in `internal/server/audit_page.go`. |
| “Einstellungen” | `/app/settings` | `internal/web/settings.templ`: `SettingsHubPage` → `SettingsFrame` → `SettingsHubContent` | **No** — the settings hub is navigational and account-focused; a photo would separate the title from the settings choices. | None; choices remain in-content links. | Settings scope/identity and settings groups. | Remove the separate settings header family and role pill; handler in `internal/server/server.go`, renderer in `internal/server/settings_page.go`. |
| “Hilfe” | `/app/hilfe` | `internal/web/help.templ`: `HelpPage` | **No** — connector status and recovery guidance should be visible immediately, not after an identity image. | None; filled “Connector einrichten” remains an in-content next step. | Connector mode/status badge and diagnostic content. | Replace `help-head`/`help-eyebrow` as chrome while retaining help content. Handler in `internal/server/help.go`. |

The “Verwaltung” navigation marker is currently emitted only when parking, handovers, or user management is visible; audit, settings, and help do not participate in that condition. That navigation rule is separate from chrome extraction and should not be changed in this polish run.

## Scoped unification work

1. In `internal/web/portal.templ`, extract `PortalShell`, `PortalSectionLanding`, `PortalSectionHero`, `PortalSectionHeader`, `PortalSectionIdentity`, and the named slots above. Move all shell, header, hero, identity/crumb, action-alignment, and responsive chrome CSS beside those partials.
2. Refactor “Hausüberblick” first as a behavior- and pixel-preserving extraction: `PortalHeroGreeting` becomes content inside the shared `PortalSectionHero`, with the existing dense/calm/mobile output retained.
3. Migrate “Termine” and “Aushang” to the shared hero configuration. Delete the `events.templ` hero markup/CSS clone and route-local header action styling; both headers use the shared outline action slot.
4. Migrate every no-hero landing to the same `PortalSectionLanding` + `PortalSectionHeader`: energy, contacts, documents, issues/board, ballots, parking, handovers, users, audit, settings, and help. `SettingsFrame` composes the kit; `EnergyModeStrip` plugs into `status`.
5. Delete page-owned `.home-hero`, `.page-head`, `.page-top`, `.page-heading`, `.crumb`, header `.lede`, header action, shell, sidebar, and mobile-header copies. Keep only feature-body CSS in each route template.
6. Do not normalize cards, tables, dialogs, forms, content grids, feature controls, workflows, permissions, routes, or visitor copy. This run is component extraction and composition, not a page rewrite.
7. Extend `internal/server/portal_chrome_test.go` with all 14 landing routes and add geometry checks to `scripts/snapshot/qa-main-flows.mjs`: shared component presence, hero decision, zero top gap, action treatment, sidebar invariants, mobile role placement, and retained energy-strip wrapping.

## Non-regression contract

- **“Hausüberblick” remains the visual baseline.** Preserve its photo crop, gradient, hairline, 150px desktop band, greeting, metrics, action position, dense/calm choice, and mobile behavior.
- **HAUSV-558 remains intact.** Do not change `EnergyModeStrip` flex wrapping, peak-chip collapse, container queries, sticky behavior, or mobile offsets in `internal/web/energy.templ`.
- **The sidebar identity remains intact.** Keep the flush map supplied by `portalMapForTenant`, one small address, and the quiet portal dropdown in `PortalSidebar`; do not duplicate them in section headers or heroes.
- **Role belongs only in the account footer.** The target shared kit renders it in `.account small`, not in `PortalMobileHeader`, portal-switch rows, page identity, or the settings body. The current mobile/context/settings leaks should be removed during composition.
- Do not change handlers, route gates, module visibility, capabilities, data queries, forms, or action behavior while migrating templates.

No UI implementation, merge, or version bump is part of this analysis branch.
