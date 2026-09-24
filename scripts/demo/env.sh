# Deterministic local demo fixture for the Hausverwaltung Musterstadt scenario.
# Start it from the repository root with: HV_DEV_FIXTURE=demo scripts/dev.sh
# Rebuild committed seed data with: go run ./scripts/demo/gen -out scripts/demo/seed

demo_repo=${repo:-$(git rev-parse --show-toplevel)}
export DEMO_SEED_DIR="$demo_repo/scripts/demo/seed"
# Inherit the isolated local services and safe integration defaults first.
# shellcheck source=scripts/snapshot/env.sh
. "$demo_repo/scripts/snapshot/env.sh"

export WEG_ORGANISATIONS_JSON='{"musterstadt":{"name":"Hausverwaltung Musterstadt GmbH"}}'
export DEFAULT_TENANT=janusbergweg-123
export WEG_TENANTS_JSON='[{"slug":"demo","name":"Demohaus","address":"Musterweg 1, 1010 Wien","map_latitude":48.2082,"map_longitude":16.3738,"map_zoom":17},{"slug":"haus-a","name":"Haus A","address":"Beispielweg 2, 1020 Wien"},{"slug":"haus-b","name":"Haus B","address":"Beispielweg 3, 1030 Wien","map_latitude":47.0707,"map_longitude":15.4395,"map_zoom":17},{"slug":"cockpit","name":"Energiehaus","address":"Energiestraße 8, 1020 Wien"},{"slug":"janusbergweg-123","name":"Janusbergweg 123","address":"Janusbergweg 123, 8010 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.0894,"map_longitude":15.4387,"map_zoom":17},{"slug":"grazbachgasse-14","name":"Grazbachgasse 14","address":"Grazbachgasse 14, 8010 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.0639,"map_longitude":15.4438,"map_zoom":17},{"slug":"annenstrasse-71","name":"Annenstraße 71","address":"Annenstraße 71, 8020 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.0716,"map_longitude":15.4221,"map_zoom":17},{"slug":"muenzgrabenstrasse-9","name":"Münzgrabenstraße 9","address":"Münzgrabenstraße 9, 8010 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.0619,"map_longitude":15.4497,"map_zoom":17},{"slug":"schoergelgasse-25","name":"Schörgelgasse 25","address":"Schörgelgasse 25, 8010 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.0654,"map_longitude":15.4537,"map_zoom":17},{"slug":"leonhardstrasse-3","name":"Leonhardstraße 3","address":"Leonhardstraße 3, 8010 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.0752,"map_longitude":15.4505,"map_zoom":17},{"slug":"koerblergasse-40","name":"Körblergasse 40","address":"Körblergasse 40, 8010 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.0807,"map_longitude":15.4452,"map_zoom":17},{"slug":"elisabethstrasse-12","name":"Elisabethstraße 12","address":"Elisabethstraße 12, 8010 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.0729,"map_longitude":15.4502,"map_zoom":17},{"slug":"sparbersbachgasse-58","name":"Sparbersbachgasse 58","address":"Sparbersbachgasse 58, 8010 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.0611,"map_longitude":15.4601,"map_zoom":17},{"slug":"mariatroster-strasse-101","name":"Mariatroster Straße 101","address":"Mariatroster Straße 101, 8043 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.1054,"map_longitude":15.4748,"map_zoom":17},{"slug":"petersgasse-7","name":"Petersgasse 7","address":"Petersgasse 7, 8010 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.0585,"map_longitude":15.4494,"map_zoom":17},{"slug":"kaiserfeldgasse-19","name":"Kaiserfeldgasse 19","address":"Kaiserfeldgasse 19, 8010 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.0678,"map_longitude":15.4388,"map_zoom":17},{"slug":"musterstrasse-12","name":"Musterstraße 12 · Zinshaus","address":"Musterstraße 12, 8010 Graz","organisation":"musterstadt","portal_type":"community","map_latitude":47.0708,"map_longitude":15.4412,"map_zoom":17}]'

demo_houses='["janusbergweg-123","grazbachgasse-14","annenstrasse-71","muenzgrabenstrasse-9","schoergelgasse-25","leonhardstrasse-3","koerblergasse-40","elisabethstrasse-12","sparbersbachgasse-58","mariatroster-strasse-101","petersgasse-7","kaiserfeldgasse-19","musterstrasse-12"]'
WEG_USERS_JSON='[{"email":"admin@example.com","first_name":"Ada","last_name":"Admin","role":"Admin","status":"Aktiv","tenants":["demo"],"auth_methods":["email"]}'

demo_add_user() {
    demo_email=$1
    demo_first=$2
    demo_last=$3
    demo_role=$4
    demo_house=$5
    demo_phone=$6
    WEG_USERS_JSON="$WEG_USERS_JSON,{\"email\":\"$demo_email\",\"first_name\":\"$demo_first\",\"last_name\":\"$demo_last\",\"phone\":\"$demo_phone\",\"role\":\"$demo_role\",\"status\":\"Aktiv\",\"tenants\":[\"$demo_house\"],\"auth_methods\":[\"email\"]}"
}

WEG_USERS_JSON="$WEG_USERS_JSON,{\"email\":\"vera.verwalter@musterstadt.example\",\"first_name\":\"Vera\",\"last_name\":\"Verwalter\",\"phone\":\"+43 316 555 100\",\"role\":\"Admin\",\"status\":\"Aktiv\",\"tenants\":$demo_houses,\"auth_methods\":[\"email\"]}"
WEG_USERS_JSON="$WEG_USERS_JSON,{\"email\":\"paul.verwalter@musterstadt.example\",\"first_name\":\"Paul\",\"last_name\":\"Sommer\",\"phone\":\"+43 316 555 101\",\"role\":\"Verwalter\",\"status\":\"Aktiv\",\"tenants\":$demo_houses,\"auth_methods\":[\"email\"]}"

demo_add_user alina.eigentuemer@musterstadt.example Alina Auer Eigentümer janusbergweg-123 "+43 664 310 20 01"
demo_add_user matthias.mieter@musterstadt.example Matthias Dorn Mieter janusbergweg-123 "+43 676 310 20 02"
demo_add_user sophie.bewohner@musterstadt.example Sophie Berger Bewohner janusbergweg-123 "+43 660 310 20 03"
demo_add_user hedwig.beirat@musterstadt.example Hedwig Eder Beirat janusbergweg-123 "+43 664 310 20 37"
demo_add_user theresa.fink@example.example Theresa Fink Eigentümer grazbachgasse-14 "+43 664 310 20 04"
demo_add_user gregor.haas@example.example Gregor Haas Mieter grazbachgasse-14 "+43 676 310 20 05"
demo_add_user nora.illek@example.example Nora Illek Bewohner grazbachgasse-14 "+43 660 310 20 06"
demo_add_user ivan.gruber@example.example Ivan Gruber Beirat grazbachgasse-14 "+43 676 310 20 38"
demo_add_user jasmin.kern@example.example Jasmin Kern Eigentümer annenstrasse-71 "+43 664 310 20 07"
demo_add_user lorenz.leitner@example.example Lorenz Leitner Mieter annenstrasse-71 "+43 676 310 20 08"
demo_add_user mira.moser@example.example Mira Moser Bewohner annenstrasse-71 "+43 660 310 20 09"
demo_add_user marlene.karner@example.example Marlene Karner Beirat annenstrasse-71 "+43 660 310 20 39"
demo_add_user daniel.novak@example.example Daniel Novak Eigentümer muenzgrabenstrasse-9 "+43 664 310 20 10"
demo_add_user petra.ortner@example.example Petra Ortner Mieter muenzgrabenstrasse-9 "+43 676 310 20 11"
demo_add_user ramin.pichler@example.example Ramin Pichler Bewohner muenzgrabenstrasse-9 "+43 660 310 20 12"
demo_add_user oskar.lind@example.example Oskar Lind Beirat muenzgrabenstrasse-9 "+43 664 310 20 40"
demo_add_user elisa.rauch@example.example Elisa Rauch Eigentümer schoergelgasse-25 "+43 664 310 20 13"
demo_add_user simon.schober@example.example Simon Schober Mieter schoergelgasse-25 "+43 676 310 20 14"
demo_add_user leyla.tas@example.example Leyla Tas Bewohner schoergelgasse-25 "+43 660 310 20 15"
demo_add_user valentin.unger@example.example Valentin Unger Eigentümer leonhardstrasse-3 "+43 664 310 20 16"
demo_add_user carina.wolf@example.example Carina Wolf Mieter leonhardstrasse-3 "+43 676 310 20 17"
demo_add_user hannes.zeller@example.example Hannes Zeller Bewohner leonhardstrasse-3 "+43 660 310 20 18"
demo_add_user bettina.almer@example.example Bettina Almer Eigentümer koerblergasse-40 "+43 664 310 20 19"
demo_add_user emir.basic@example.example Emir Basic Mieter koerblergasse-40 "+43 676 310 20 20"
demo_add_user klara.cerny@example.example Klara Cerny Bewohner koerblergasse-40 "+43 660 310 20 21"
demo_add_user david.ebner@example.example David Ebner Eigentümer elisabethstrasse-12 "+43 664 310 20 22"
demo_add_user fatma.guel@example.example Fatma Gül Mieter elisabethstrasse-12 "+43 676 310 20 23"
demo_add_user josef.hofer@example.example Josef Hofer Bewohner elisabethstrasse-12 "+43 660 310 20 24"
demo_add_user iris.jauk@example.example Iris Jauk Eigentümer sparbersbachgasse-58 "+43 664 310 20 25"
demo_add_user kemal.kaya@example.example Kemal Kaya Mieter sparbersbachgasse-58 "+43 676 310 20 26"
demo_add_user lena.lenz@example.example Lena Lenz Bewohner sparbersbachgasse-58 "+43 660 310 20 27"
demo_add_user moritz.maier@example.example Moritz Maier Eigentümer mariatroster-strasse-101 "+43 664 310 20 28"
demo_add_user nadine.oswald@example.example Nadine Oswald Mieter mariatroster-strasse-101 "+43 676 310 20 29"
demo_add_user peter.reiter@example.example Peter Reiter Bewohner mariatroster-strasse-101 "+43 660 310 20 30"
demo_add_user selma.sari@example.example Selma Sari Eigentümer petersgasse-7 "+43 664 310 20 31"
demo_add_user tobias.thaler@example.example Tobias Thaler Mieter petersgasse-7 "+43 676 310 20 32"
demo_add_user ulrike.weiss@example.example Ulrike Weiss Bewohner petersgasse-7 "+43 660 310 20 33"
demo_add_user florian.zechner@example.example Florian Zechner Eigentümer kaiserfeldgasse-19 "+43 664 310 20 34"
demo_add_user gerlinde.binder@example.example Gerlinde Binder Mieter kaiserfeldgasse-19 "+43 676 310 20 35"
demo_add_user armin.cakir@example.example Armin Cakir Bewohner kaiserfeldgasse-19 "+43 660 310 20 36"

WEG_USERS_JSON="$WEG_USERS_JSON]"
export WEG_USERS_JSON
export ADMIN_EMAILS=admin@example.com
INVITE_EMAILS='admin@example.com,vera.verwalter@musterstadt.example,paul.verwalter@musterstadt.example,alina.eigentuemer@musterstadt.example,matthias.mieter@musterstadt.example,sophie.bewohner@musterstadt.example,hedwig.beirat@musterstadt.example,theresa.fink@example.example,gregor.haas@example.example,nora.illek@example.example,ivan.gruber@example.example,jasmin.kern@example.example,lorenz.leitner@example.example,mira.moser@example.example,marlene.karner@example.example,daniel.novak@example.example,petra.ortner@example.example,ramin.pichler@example.example,oskar.lind@example.example,elisa.rauch@example.example,simon.schober@example.example,leyla.tas@example.example,valentin.unger@example.example,carina.wolf@example.example,hannes.zeller@example.example,bettina.almer@example.example,emir.basic@example.example,klara.cerny@example.example,david.ebner@example.example,fatma.guel@example.example,josef.hofer@example.example,iris.jauk@example.example,kemal.kaya@example.example,lena.lenz@example.example,moritz.maier@example.example,nadine.oswald@example.example,peter.reiter@example.example,selma.sari@example.example,tobias.thaler@example.example,ulrike.weiss@example.example,florian.zechner@example.example,gerlinde.binder@example.example,armin.cakir@example.example'
export INVITE_EMAILS

unset demo_email demo_first demo_last demo_role demo_house demo_phone demo_houses
unset -f demo_add_user
