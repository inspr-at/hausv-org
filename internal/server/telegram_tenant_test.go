package server

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/homeassistant"
	"github.com/inspr-at/hausv-org/internal/store"
)

// HAUSV-813: a linked chat may control only a house the linked user is
// allowed to control at the moment of the command. The default house is not
// a target.

func addTelegramChargingHouse(t *testing.T, a *app, slug, name string, ha *fakeHA) {
	t.Helper()
	cfg := homeassistant.NewConfig(ha.srv.URL, "test-token", "sensor.meter", "sensor.power", "sensor.price").
		WithChargingEntities("switch.plug", "sensor.soc", "sensor.feed")
	a.tenants[slug] = tenantConfig{Slug: slug, Name: name, HA: cfg}
	if err := a.parkingStore.SetChargingControl(slug, chargingControlSettings{Enabled: true}); err != nil {
		t.Fatal(err)
	}
}

func renameTelegramHouse(a *app, slug, name string) {
	tenant := a.tenants[slug]
	tenant.Slug = slug
	tenant.Name = name
	a.tenants[slug] = tenant
}

func TestTelegramHouseAPermissionCannotControlB(t *testing.T) {
	a, haB, _ := newTelegramTestApp(t)
	renameTelegramHouse(a, "demo", "Haus B")
	haA := newFakeHA(t)
	addTelegramChargingHouse(t, a, "haus-a", "Haus A", haA)
	a.defaultTenant = "demo"
	a.profiles["ada@example.com"] = userProfile{
		Email:       "ada@example.com",
		Role:        roleResident,
		Tenants:     []string{"haus-a"},
		Permissions: []string{permissionParking},
	}
	linkChat(t, a, 8101, "ada@example.com")

	reply := a.handleTelegramCommand(context.Background(), 8101, "Ada", "/pp20ein")
	if sessions := a.parkingStore.TenantData("demo").ChargingSessions; len(sessions) != 0 {
		t.Fatalf("house B was controlled: sessions=%+v reply=%q", sessions, reply)
	}
	if calls := haB.calls(); len(calls) != 0 {
		t.Fatalf("house B plug switched: %v reply=%q", calls, reply)
	}
	if sessions := a.parkingStore.TenantData("haus-a").ChargingSessions; len(sessions) != 1 {
		t.Fatalf("house A was not controlled: sessions=%+v reply=%q", sessions, reply)
	}
	if calls := haA.calls(); len(calls) != 1 || calls[0] != "turn_on" {
		t.Fatalf("house A plug = %v", calls)
	}
	if !strings.Contains(reply, "Haus A") || strings.Contains(reply, "Haus B") {
		t.Fatalf("reply must name only house A: %q", reply)
	}
}

func TestTelegramAdminOfACannotControlB(t *testing.T) {
	a, haB, _ := newTelegramTestApp(t)
	renameTelegramHouse(a, "demo", "Haus B")
	haA := newFakeHA(t)
	addTelegramChargingHouse(t, a, "haus-a", "Haus A", haA)
	a.defaultTenant = "demo"
	a.profiles["ada@example.com"] = userProfile{
		Email:   "ada@example.com",
		Role:    roleAdmin,
		Tenants: []string{"haus-a"},
	}
	linkChat(t, a, 8102, "ada@example.com")

	reply := a.handleTelegramCommand(context.Background(), 8102, "Ada", "/pp20ein")
	if len(a.parkingStore.TenantData("demo").ChargingSessions) != 0 || len(haB.calls()) != 0 {
		t.Fatalf("admin of A controlled B: reply=%q callsB=%v", reply, haB.calls())
	}
	if len(a.parkingStore.TenantData("haus-a").ChargingSessions) != 1 {
		t.Fatalf("admin of A did not control A: reply=%q", reply)
	}
}

func TestTelegramMembershipPermissionDoesNotCrossHouses(t *testing.T) {
	a, haB, _ := newTelegramTestApp(t)
	renameTelegramHouse(a, "demo", "Haus B")
	haA := newFakeHA(t)
	addTelegramChargingHouse(t, a, "haus-a", "Haus A", haA)
	a.defaultTenant = "demo"
	a.profiles["ada@example.com"] = userProfile{
		Email:   "ada@example.com",
		Role:    roleResident,
		Tenants: []string{"haus-a", "demo"},
		TenantMemberships: map[string]tenantMembership{
			"haus-a": {Role: roleResident, Permissions: []string{permissionParking}},
			"demo":   {Role: roleResident, Permissions: []string{}},
		},
	}
	linkChat(t, a, 8103, "ada@example.com")

	reply := a.handleTelegramCommand(context.Background(), 8103, "Ada", "/pp20ein")
	if len(a.parkingStore.TenantData("demo").ChargingSessions) != 0 || len(haB.calls()) != 0 {
		t.Fatalf("membership on A controlled B: reply=%q", reply)
	}
	if len(a.parkingStore.TenantData("haus-a").ChargingSessions) != 1 || !strings.Contains(reply, "Haus A") {
		t.Fatalf("membership on A did not control A: reply=%q", reply)
	}
}

func TestTelegramAmbiguousHousePromptsAndSessionSelection(t *testing.T) {
	a, haB, _ := newTelegramTestApp(t)
	renameTelegramHouse(a, "demo", "Haus B")
	haA := newFakeHA(t)
	addTelegramChargingHouse(t, a, "haus-a", "Haus A", haA)
	a.defaultTenant = "demo"
	a.profiles["ada@example.com"] = userProfile{
		Email:       "ada@example.com",
		Role:        roleResident,
		Tenants:     []string{"haus-a", "demo"},
		Permissions: []string{permissionParking},
	}
	storePath := filepath.Join(t.TempDir(), "telegram.json")
	tgStore, err := store.NewTelegramStore(storePath)
	if err != nil {
		t.Fatal(err)
	}
	a.telegramStore = tgStore
	linkChat(t, a, 8104, "ada@example.com")

	reply := a.handleTelegramCommand(context.Background(), 8104, "Ada", "/pp20ein")
	if len(haA.calls()) != 0 || len(haB.calls()) != 0 {
		t.Fatalf("ambiguous command switched a plug: A=%v B=%v reply=%q", haA.calls(), haB.calls(), reply)
	}
	if len(a.parkingStore.TenantData("haus-a").ChargingSessions) != 0 || len(a.parkingStore.TenantData("demo").ChargingSessions) != 0 {
		t.Fatal("ambiguous command started a session")
	}
	for _, want := range []string{"/haus", "Haus A", "Haus B"} {
		if !strings.Contains(reply, want) {
			t.Fatalf("prompt %q missing %q", reply, want)
		}
	}
	for _, leak := range []string{"Steckdose", "Akku", "Zähler", "Überschuss", "eingeschaltet"} {
		if strings.Contains(reply, leak) {
			t.Fatalf("prompt disclosed charging state %q in %q", leak, reply)
		}
	}

	chosen := a.handleTelegramCommand(context.Background(), 8104, "Ada", "/haus Haus A")
	if !strings.Contains(chosen, "Haus A") || strings.Contains(chosen, "Haus B") {
		t.Fatalf("selection reply = %q", chosen)
	}
	if len(haA.calls()) != 0 || len(haB.calls()) != 0 {
		t.Fatal("/haus must not switch a plug")
	}

	acted := a.handleTelegramCommand(context.Background(), 8104, "Ada", "/pp20ein")
	if len(haA.calls()) != 1 || len(haB.calls()) != 0 {
		t.Fatalf("selection targeted the wrong house: A=%v B=%v reply=%q", haA.calls(), haB.calls(), acted)
	}
	if !strings.Contains(acted, "Haus A") || strings.Contains(acted, "Haus B") {
		t.Fatalf("action reply = %q", acted)
	}

	// The choice lives only in this process. The telegram store must not gain a house.
	raw, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "haus-a") || strings.Contains(string(raw), "Haus A") {
		t.Fatalf("selection was persisted: %s", raw)
	}
}

func TestTelegramRevokedPermissionDeniedOnNextCommand(t *testing.T) {
	a, _, _ := newTelegramTestApp(t)
	renameTelegramHouse(a, "demo", "Haus B")
	haA := newFakeHA(t)
	addTelegramChargingHouse(t, a, "haus-a", "Haus A", haA)
	a.profiles["ada@example.com"] = userProfile{
		Email:       "ada@example.com",
		Role:        roleResident,
		Tenants:     []string{"haus-a"},
		Permissions: []string{permissionParking},
	}
	linkChat(t, a, 8105, "ada@example.com")

	first := a.handleTelegramCommand(context.Background(), 8105, "Ada", "/pp20ein")
	if len(haA.calls()) != 1 || !strings.Contains(first, "Haus A") {
		t.Fatalf("first command = %q calls=%v", first, haA.calls())
	}

	profile := a.profiles["ada@example.com"]
	profile.Permissions = nil
	profile.TenantMemberships = map[string]tenantMembership{
		"haus-a": {Role: roleResident, Permissions: []string{}},
	}
	a.profiles["ada@example.com"] = profile

	second := a.handleTelegramCommand(context.Background(), 8105, "Ada", "/pp20ein")
	if !strings.Contains(second, "Berechtigung") {
		t.Fatalf("revoked reply = %q", second)
	}
	if len(haA.calls()) != 1 {
		t.Fatalf("revoked permission still switched the plug: %v", haA.calls())
	}
	if strings.Contains(second, "Haus A") || strings.Contains(second, "Haus B") || strings.Contains(second, "Steckdose") {
		t.Fatalf("denial disclosed a house or a state: %q", second)
	}
}

func TestTelegramStaleHouseSelectionIsNotAuthority(t *testing.T) {
	a, haB, _ := newTelegramTestApp(t)
	renameTelegramHouse(a, "demo", "Haus B")
	haA := newFakeHA(t)
	addTelegramChargingHouse(t, a, "haus-a", "Haus A", haA)
	a.profiles["ada@example.com"] = userProfile{
		Email:       "ada@example.com",
		Role:        roleResident,
		Tenants:     []string{"haus-a", "demo"},
		Permissions: []string{permissionParking},
	}
	linkChat(t, a, 8106, "ada@example.com")
	if reply := a.handleTelegramCommand(context.Background(), 8106, "Ada", "/haus Haus A"); !strings.Contains(reply, "Haus A") {
		t.Fatalf("select = %q", reply)
	}
	if reply := a.handleTelegramCommand(context.Background(), 8106, "Ada", "/pp20ein"); len(haA.calls()) != 1 || len(haB.calls()) != 0 {
		t.Fatalf("selected A did not stick: A=%v B=%v reply=%q", haA.calls(), haB.calls(), reply)
	}

	profile := a.profiles["ada@example.com"]
	profile.Permissions = nil
	profile.TenantMemberships = map[string]tenantMembership{
		"haus-a": {Role: roleResident, Permissions: []string{}},
		"demo":   {Role: roleResident, Permissions: []string{permissionParking}},
	}
	a.profiles["ada@example.com"] = profile

	reply := a.handleTelegramCommand(context.Background(), 8106, "Ada", "/pp20ein")
	if len(haA.calls()) != 1 {
		t.Fatalf("revoked A was switched again: %v reply=%q", haA.calls(), reply)
	}
	if len(haB.calls()) != 1 || !strings.Contains(reply, "Haus B") || strings.Contains(reply, "Haus A") {
		t.Fatalf("remaining house B was not used: B=%v reply=%q", haB.calls(), reply)
	}
}

func TestTelegramUnlinkedChatDisclosesNothing(t *testing.T) {
	a, _, _ := newTelegramTestApp(t)
	renameTelegramHouse(a, "demo", "Haus B")
	haA := newFakeHA(t)
	addTelegramChargingHouse(t, a, "haus-a", "Haus A", haA)

	for _, text := range []string{"/pp20ein", "/pp20status", "/haus Haus A", "/PP20EIN@HausvOrgBot", "hello"} {
		reply := a.handleTelegramCommand(context.Background(), 8107, "Ada", text)
		if !strings.Contains(reply, "nicht verknüpft") {
			t.Fatalf("reply for %q = %q", text, reply)
		}
		for _, leak := range []string{"Haus A", "Haus B", "haus-a", "demo", "Parkplatz", "Steckdose", "Akku", "freigegeben", "Test"} {
			if strings.Contains(reply, leak) {
				t.Fatalf("unlinked reply for %q disclosed %q: %q", text, leak, reply)
			}
		}
	}
	start := a.handleTelegramCommand(context.Background(), 8107, "Ada", "/start")
	if !strings.Contains(start, "nicht verknüpft") || !strings.Contains(start, "Code") {
		t.Fatalf("start = %q", start)
	}
	for _, leak := range []string{"Haus A", "Haus B", "haus-a", "Parkplatz", "Steckdose"} {
		if strings.Contains(start, leak) {
			t.Fatalf("unlinked /start disclosed %q: %q", leak, start)
		}
	}
}

func TestTelegramPP20AliasesStayCompatible(t *testing.T) {
	a, ha, _ := newTelegramTestApp(t)
	renameTelegramHouse(a, "demo", "Haus Ada")
	linkChat(t, a, joergChat, "joerg@example.com")

	on := a.handleTelegramCommand(context.Background(), joergChat, "Jörg", "/PP20EIN@HausvOrgBot")
	if !strings.Contains(on, "eingeschaltet") || !strings.Contains(on, "Haus Ada") {
		t.Fatalf("alias on = %q", on)
	}
	if calls := ha.calls(); len(calls) != 1 || calls[0] != "turn_on" {
		t.Fatalf("alias did not switch: %v", calls)
	}
	status := a.handleTelegramCommand(context.Background(), joergChat, "Jörg", "/PP20STATUS@HausvOrgBot")
	if !strings.Contains(status, "Parkplatz 20") || !strings.Contains(status, "Manuell eingeschaltet") || !strings.Contains(status, "Haus Ada") {
		t.Fatalf("alias status = %q", status)
	}
	off := a.handleTelegramCommand(context.Background(), joergChat, "Jörg", "/pp20aus")
	if !strings.Contains(off, "warten") {
		t.Fatalf("alias off = %q", off)
	}
	auto := a.handleTelegramCommand(context.Background(), joergChat, "Jörg", "/pp20auto")
	if auto == "" || !strings.Contains(auto, "Haus Ada") {
		t.Fatalf("alias auto = %q", auto)
	}
}

func TestTelegramCommandAuditLine(t *testing.T) {
	previous := slog.Default()
	var logs strings.Builder
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	a, _, _ := newTelegramTestApp(t)
	renameTelegramHouse(a, "demo", "Haus B")
	haA := newFakeHA(t)
	addTelegramChargingHouse(t, a, "haus-a", "Haus A", haA)
	a.profiles["ada@example.com"] = userProfile{
		Email:       "ada@example.com",
		Role:        roleResident,
		Tenants:     []string{"haus-a"},
		Permissions: []string{permissionParking},
	}
	linkChat(t, a, 8108, "ada@example.com")
	logs.Reset()

	reply := a.handleTelegramCommand(context.Background(), 8108, "Ada", "/pp20ein")
	if !strings.Contains(reply, "Haus A") {
		t.Fatalf("reply = %q", reply)
	}
	line := telegramAuditLine(t, logs.String(), "/pp20ein")
	for _, want := range []string{`"user":"a***@example.com"`, `"house":"haus-a"`, `"command":"/pp20ein"`, `"outcome":"ok"`} {
		if !strings.Contains(line, want) {
			t.Fatalf("audit line %s missing %s", line, want)
		}
	}
	if strings.Contains(line, "ada@example.com") || strings.Contains(logs.String(), "test-token") {
		t.Fatalf("audit leaked a secret or the full address: %s", logs.String())
	}

	logs.Reset()
	_ = a.handleTelegramCommand(context.Background(), 8109, "Ada", "/start GEHEIMCODE")
	logged := logs.String()
	if strings.Contains(logged, "GEHEIMCODE") {
		t.Fatalf("link code was logged: %s", logged)
	}
	rejected := telegramAuditLine(t, logged, "/start")
	if !strings.Contains(rejected, `"outcome":"link_rejected"`) || !strings.Contains(rejected, `"house":""`) {
		t.Fatalf("rejected link audit = %s", rejected)
	}
}

func TestTelegramDuplicateHouseNamesAreAddressedBySlug(t *testing.T) {
	a, haB, _ := newTelegramTestApp(t)
	renameTelegramHouse(a, "demo", "Gleiche")
	haA := newFakeHA(t)
	addTelegramChargingHouse(t, a, "haus-a", "Gleiche", haA)
	a.profiles["ada@example.com"] = userProfile{
		Email:       "ada@example.com",
		Role:        roleResident,
		Tenants:     []string{"haus-a", "demo"},
		Permissions: []string{permissionParking},
	}
	linkChat(t, a, 8110, "ada@example.com")

	prompt := a.handleTelegramCommand(context.Background(), 8110, "Ada", "/pp20ein")
	if !strings.Contains(prompt, "/haus haus-a") || !strings.Contains(prompt, "/haus demo") {
		t.Fatalf("prompt = %q", prompt)
	}
	if len(haA.calls()) != 0 || len(haB.calls()) != 0 {
		t.Fatal("prompt switched a plug")
	}
	chosen := a.handleTelegramCommand(context.Background(), 8110, "Ada", "/haus haus-a")
	if !strings.Contains(chosen, "haus-a") || strings.Contains(chosen, "demo") {
		t.Fatalf("selection = %q", chosen)
	}
	acted := a.handleTelegramCommand(context.Background(), 8110, "Ada", "/pp20ein")
	if len(haA.calls()) != 1 || len(haB.calls()) != 0 || !strings.Contains(acted, "haus-a") {
		t.Fatalf("acted A=%v B=%v reply=%q", haA.calls(), haB.calls(), acted)
	}
}

func telegramAuditLine(t *testing.T, logged, command string) string {
	t.Helper()
	for _, line := range strings.Split(logged, "\n") {
		if strings.Contains(line, "Telegram command") && strings.Contains(line, `"command":"`+command+`"`) {
			return line
		}
	}
	t.Fatalf("no audit line for %s in %s", command, logged)
	return ""
}
