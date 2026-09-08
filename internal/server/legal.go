package server

const (
	platformContactEmail = "hello@hausv.org"
	legalReviewDate      = "14. August 2026"
)

func platformOperatorName() string {
	return env("HAUSV_OPERATOR_NAME", "Betreiber laut Host-Konfiguration")
}

func platformOperatorAddress() string {
	return env("HAUSV_OPERATOR_ADDRESS", "Anschrift laut Host-Konfiguration")
}

func primaryAppURL() string {
	return env("HAUSV_PRIMARY_APP_URL", "https://hausv.org/demo/")
}

func professionalServicesNotice() string {
	return env("HAUSV_PROFESSIONAL_SERVICES_NOTICE", "Professionelle Einrichtung und Betreuung sind über HAUSV Professional verfügbar.")
}

func identityStorageNotice() string {
	return env("HAUSV_IDENTITY_STORAGE_NOTICE", "Fachdaten und Identitätsdienste werden in der vom Betreiber dokumentierten Infrastruktur verarbeitet und je Liegenschaft und Rolle getrennt.")
}

func backupStorageNotice() string {
	return env("HAUSV_BACKUP_STORAGE_NOTICE", "Sicherungen werden verschlüsselt und gemäß der dokumentierten Betreiberkonfiguration aufbewahrt.")
}

func webAccessNotice() string {
	return env("HAUSV_WEB_ACCESS_NOTICE", "Der öffentliche Webzugriff wird über die vom Betreiber dokumentierte Infrastruktur vermittelt und geschützt.")
}

func mailDeliveryNotice() string {
	return env("HAUSV_MAIL_DELIVERY_NOTICE", "Transaktionsmails werden über den vom Betreiber dokumentierten Maildienst versendet.")
}
