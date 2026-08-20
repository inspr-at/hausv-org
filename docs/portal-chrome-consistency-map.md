# Portal left-navigation chrome map

Baseline: HAUSV 0.98.11, tenant `jhw22`. This is an implementation map and a scoped polish recommendation; it does not propose product or interaction changes.

## Route and template inventory

The source navigation uses canonical `/app...` paths. `tenantPaths` and `prefixTenantHTMLPaths` in `internal/server/server.go` expose those paths as `/jhw22/app...` on the live tenant.

| Menu label | Route | Templ file | Chrome family | Same as | Notes |
| --- | --- | --- | --- | --- | --- |
| “Hausüberblick” | `/app` | `internal/web/portal.templ`: `PortalPage` → `PortalContent` → `PortalDense` or `PortalCalm`; `PortalMobile` on mobile | Home hero | Approved baseline | `portal` handler in `internal/server/server.go`. `PortalHeroGreeting` owns the greeting markup, while `PortalStyles` owns `.home-hero`. |
| “Mein Zuhause” | `/app/energie` | `internal/web/energy.templ`: `EnergyPage` → `EnergyModeStrip`, `EnergyHeading` | Energy cockpit | No primary-nav peer; onboarding mirrors the strip | `energyCockpit` handler in `internal/server/energy.go`; renderer in `internal/server/energy_page.go`. The visible nav label can be replaced by `HomeIdentity.DisplayName`; “Mein Zuhause” is the fallback and accessible label. |
| “Aushang” | `/app/announcements` | `internal/web/announcements.templ`: `AnnouncementsPage` → `AnnouncementsContent` | Eyebrow `page-head` | “Anliegen”, “Abstimmungen”, “Parkplatznutzung”, “Verlauf” | `announcements` handler in `internal/server/announcements.go`. The heading is route-local, not a shared partial. |
| “Termine” | `/app/events` | `internal/web/events.templ`: `EventsPage` → `EventsContent` | Local `home-hero` clone plus `page-top` crumb | Intended to match “Hausüberblick”; actually standalone | `events` handler in `internal/server/events.go`. It does not call `PortalHeroGreeting`; it copies the image, gradient, hairline, and responsive CSS. |
| “Kontakte” | `/app/kontakte` | `internal/web/contacts.templ`: `ContactsPage` → `ContactsContent` | Crumb-only `page-top` plus `page-heading` | “Dokumente”, “Übergaben” | `contacts` handler in `internal/server/contacts.go`. Its mobile label passes the role into `PortalMobileHeader`, which appends the role again. |
| “Dokumente” | `/app/dokumente` | `internal/web/documents.templ`: `DocumentsPage` → `DocumentsContent` | Crumb-only `page-top` plus `page-heading` | “Kontakte”, “Übergaben” | `documents` handler in `internal/server/documents.go`. Shell, crumb, heading, and button CSS are local copies. |
| “Anliegen” | Resident: `/app/anliegen`; manager: `/app/anliegen/board` | `internal/web/issues.templ`: `IssuesPage` → `IssuesContent`; `internal/web/issue_board.templ`: `IssueBoardPage` → `IssueBoardContent` | Eyebrow `page-head` | “Aushang”, “Abstimmungen”, “Parkplatznutzung”, “Verlauf” | `issues` and `issueBoard` handlers in `internal/server/issues.go`. Manager navigation selects the board route. Triage and detail routes are drill-down flows, not the section landing. |
| “Abstimmungen” | `/app/abstimmungen` | `internal/web/ballots.templ`: `BallotsPage` | Eyebrow `page-head` | “Aushang”, “Anliegen”, “Parkplatznutzung”, “Verlauf” | `ballots` handler in `internal/server/votes.go`. The page wrapper, heading, and action treatment are route-local. |
| “Parkplatznutzung” | `/app/parking` | `internal/web/parking.templ`: `ParkingPage` → `ParkingContent` | Eyebrow `page-head` | “Aushang”, “Anliegen”, “Abstimmungen”, “Verlauf” | `parking` handler in `internal/server/parking.go`. The page title itself is “Parkplatz”. |
| “Übergaben” | `/app/uebergaben` | `internal/web/handovers.templ`: `HandoversPage` → `HandoversContent` | Crumb-only `page-top` plus `handover-page-head` | “Kontakte”, “Dokumente” | `handovers` handler in `internal/server/handover.go`. The header is a local variation of the crumb-only family. |
| “Benutzer & Rechte” | `/app/settings/users` | `internal/web/settings.templ`: `UserSettingsPage` → `SettingsFrame` → `UserSettingsContent` | Settings frame | “Einstellungen” and settings subpages | `userSettings` handler in `internal/server/server.go`; renderer in `internal/server/settings_page.go`. `SettingsFrame` is the only route-family wrapper already shared across several pages. |
| “Verlauf” | `/app/audit` | `internal/web/audit.templ`: `AuditPage` | Eyebrow `page-head` | “Aushang”, “Anliegen”, “Abstimmungen”, “Parkplatznutzung” | `auditLog` handler in `internal/server/server.go`; renderer in `internal/server/audit_page.go`. The visitor-facing title changes with audit scope. |
| “Einstellungen” | `/app/settings` | `internal/web/settings.templ`: `SettingsHubPage` → `SettingsFrame` → `SettingsHubContent` | Settings frame | “Benutzer & Rechte” and settings subpages | `settingsHub` handler in `internal/server/server.go`; renderer in `internal/server/settings_page.go`. It has its own `page-head` styling rather than a shared section header. |
| “Hilfe” | `/app/hilfe` | `internal/web/help.templ`: `HelpPage` | Bespoke help header | No peer | `helpPage` and `renderHelpPage` handlers in `internal/server/help.go`. It uses `help-head`, `help-eyebrow`, and `help-mode`, not either common heading family. |

`PortalNavigation` in `internal/web/portal.templ` is the single source for both desktop and mobile navigation. Items are permission- and module-gated. The “Verwaltung” marker is emitted only when parking, handovers, or user management is visible; audit, settings, and help do not participate in that condition.

## Shared chrome and local copies

The genuinely shared outer pieces are all in `internal/web/portal.templ`:

- `PortalDocument` composes `PortalBaseStyles`, route styles, and `PortalShellStyles`.
- `PortalSidebar` owns the map, address, portal switcher, navigation, and account footer.
- `PortalNavigation` owns menu order, labels, routes, gates, active state, and badges.
- `PortalMobileHeader` and `PortalMobileContextSwitch` own the compact navigation.
- `portalMapForTenant` in `internal/server/portal_chrome.go` supplies the sidebar map.

The approved flush map is already centralized: `.side-brand` has negative top and side margins, and `.side-map-card` is edge-to-edge inside `PortalSidebar`. The small address and quiet dropdown also already live in `side-place-copy`. They should not be recreated in a page header.

The section-level chrome is not shared:

- `PortalHeroGreeting` is shared only by the `PortalDense`, `PortalCalm`, and `PortalMobile` variants of “Hausüberblick”.
- `EventsContent` copies `.home-hero`, `.home-hero-image`, `.home-hero::after`, and `.home-hero-copy`. Its `events-main` keeps `padding: var(--space-6)` above a normal-flow `position: relative` hero, producing the rejected 24px top gap. “Hausüberblick” puts an absolute hero inside a zero-top-padding 150px row.
- The “Termine” header action calls `EventCreateButton`, which uses filled `.button.primary`; the “Hausüberblick” top action is plain `.button`, the approved outline treatment.
- `page-top`/`crumb`/`page-heading` are repeated in contacts, documents, handovers, and events.
- `page-head`/`eyebrow`/`lede` and `.button.primary` are repeated in announcements, issues, ballots, parking, audit, and settings.
- Most page templates repeat the skip link, `PortalMobileHeader`, `.shell`, and `PortalSidebar` composition. Several route style blocks also redeclare `.shell`, `.sidebar`, and mobile-menu rules already represented by `PortalShellStyles`.

This is why changing only “Termine” did not make the portal coherent: the implementation still has home-hero, local-hero, crumb-only, eyebrow, settings, energy, and help header systems.

## Approved chrome gaps in 0.98.11

The recommended work must preserve the approved “Hausüberblick” image treatment, hairline, and sidebar map. It must also preserve the explicit `EnergyModeStrip` wrapping and container-query behavior in `internal/web/energy.templ`; the existing QA script exercises that geometry.

Two current details do not satisfy “role only in the account footer”:

- `PortalMobileHeader` appends `data.Role` to every mobile page label.
- Both desktop and mobile portal switchers render each context role. `ContactsPage` additionally passes the role in its page label, so the mobile subtitle can repeat it.

`SettingsHubContent` also renders a role pill in the page body. Under the approved “role only in the account footer” rule, that pill belongs in the same cleanup.

## One recommended landing pattern

Use one shared outer pattern for every primary-nav landing: `PortalDocument` → `PortalMobileHeader` → `.shell` → `PortalSidebar` + a normal-flow `PortalSectionLanding` whose first child is `PortalSectionHero`.

Implement both new partials and their styles in `internal/web/portal.templ`:

1. `PortalSectionLanding` owns zero top gap, the common content width, page spacing, and an action slot.
2. `PortalSectionHero` owns the full-width tenant photo, the existing gradient and crop, a bottom hairline, a section-content slot, and an optional action slot. Existing titles and ledes can move through that slot without new visual vocabulary.
3. Refactor `PortalHeroGreeting` to consume the same hero image/gradient primitive without changing “Hausüberblick” dimensions or responsive behavior.
4. Keep `PortalSidebar` as the only desktop identity block. Show one small address, retain the quiet portal dropdown, remove roles from `PortalMobileHeader` and context-switch rows, and leave the role in `.account small`.

No primary-nav landing should fork the outer chrome. Content below it may remain specialized:

- “Hausüberblick” keeps `PortalDense`/`PortalCalm`; it is the visual regression baseline.
- “Mein Zuhause” keeps `EnergyModeStrip` and the cockpit content. The strip is the one justified extra chrome element because it exposes live mode and safety controls; place it without changing its wrap, sticky offsets, or container queries.
- Settings pages keep `SettingsFrame` for form and permission structure, but `SettingsFrame` should compose the shared landing/header rather than own a different header system.
- Help keeps its connector status and instructional content, but not a bespoke landing header.
- Drill-down and transactional routes such as issue triage, onboarding, settings detail forms, parking month views, and protocols may use compact back/context headers because they are not primary section landings.

Header action rule: use the plain outline `.button` for the single action in `PortalSectionHero`, matching “Hausüberblick”. Reserve filled `.button.primary` for in-content next steps, empty states, and form/dialog submission. Keep `.button.ghost` for secondary or destructive-adjacent choices. In particular, the hero instance of “Termin erstellen” must be outline; its dialog submit and body empty-state action may remain filled.

## Scoped polish run

1. Extract `PortalSectionLanding`, `PortalSectionHero`, and shared landing/button styles in `internal/web/portal.templ`; make `PortalHeroGreeting` consume the shared visual primitive with snapshot-equivalent “Hausüberblick” output.
2. Switch `internal/web/events.templ` first and delete its local hero clone. Verify a flush top edge, the existing hairline/crop, and an outline header action.
3. Switch the remaining primary landings: `announcements.templ`, `contacts.templ`, `documents.templ`, `issues.templ`, the board landing in `issue_board.templ`, `ballots.templ`, `parking.templ`, `handovers.templ`, `audit.templ`, `settings.templ`, `help.templ`, and `energy.templ`. Leave feature-body layouts and handlers unchanged.
4. Remove only duplicated chrome selectors from route styles; do not normalize cards, data layouts, workflows, copy, or feature controls.
5. Extend `internal/server/portal_chrome_test.go` with the complete landing-route matrix and add geometry assertions to `scripts/snapshot/qa-main-flows.mjs`. Keep the existing energy-strip wrap checks and add explicit “Hausüberblick” and mobile role-placement regressions.

This is a chrome extraction and migration, not a visual redesign or page rewrite.
