package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
)

const indexPublicationHistorySource = "https://www.statistik.at/fileadmin/pages/214/VPI_-_Veroeffentlichungstermine_1977_bis_2025.pdf"
const indexPublication2026Source = "https://www.statistik.at/fileadmin/pages/214/Pubtermine2026.pdf"

// First publication is preliminary. A month's final publication is the NEXT
// observation's first publication (the explicit footnote in the official table).
// Outside the evidenced calendar we fail closed; no estimated t+45 dates.
var indexFirstPublications = map[int]string{
	2021: "23.02.2021 17.03.2021 16.04.2021 19.05.2021 17.06.2021 16.07.2021 18.08.2021 17.09.2021 20.10.2021 17.11.2021 17.12.2021 20.01.2022",
	2022: "23.02.2022 17.03.2022 21.04.2022 18.05.2022 17.06.2022 19.07.2022 18.08.2022 16.09.2022 19.10.2022 17.11.2022 16.12.2022 18.01.2023",
	2023: "23.02.2023 17.03.2023 19.04.2023 17.05.2023 16.06.2023 19.07.2023 18.08.2023 19.09.2023 18.10.2023 17.11.2023 19.12.2023 17.01.2024",
	2024: "22.02.2024 18.03.2024 17.04.2024 17.05.2024 18.06.2024 17.07.2024 20.08.2024 18.09.2024 17.10.2024 19.11.2024 18.12.2024 15.01.2025",
	2025: "24.02.2025 19.03.2025 16.04.2025 19.05.2025 18.06.2025 17.07.2025 20.08.2025 17.09.2025 17.10.2025 19.11.2025 17.12.2025 19.01.2026",
	2026: "25.02.2026 18.03.2026 16.04.2026 20.05.2026 17.06.2026 17.07.2026 19.08.2026 17.09.2026 16.10.2026 18.11.2026 17.12.2026 19.01.2027",
}

func finalIndexPublication(month indexation.Month) (time.Time, string, bool) {
	date, err := time.Parse("2006-01", string(month))
	if err != nil {
		return time.Time{}, "", false
	}
	next := date.AddDate(0, 1, 0)
	dates := strings.Fields(indexFirstPublications[next.Year()])
	if len(dates) != 12 {
		return time.Time{}, "", false
	}
	final, err := time.Parse("02.01.2006", dates[int(next.Month())-1])
	source := indexPublicationHistorySource
	if next.Year() == 2026 {
		source = indexPublication2026Source
	}
	return final, source, err == nil
}

func indexEvidence(v indexation.IndexValue) ValorisationIndex {
	pub, source, ok := indexValuePublication(v)
	date := ""
	if ok {
		date = pub.Format(time.DateOnly)
	}
	status := "final"
	if v.Preliminary {
		status = "preliminary"
	}
	if v.ChainSource != "" {
		status = "derived"
	}
	return ValorisationIndex{RevisionID: v.RevisionID, Series: string(v.Series), Period: string(v.Month), Value: v.Value.String(), Status: status, PublishedOn: date, Source: v.Source, PublicationSource: source}
}

func annualEvidence(v indexation.AnnualValue) ValorisationIndex {
	pub, source, ok := indexValuePublication(indexation.IndexValue{Month: indexation.Month(fmt.Sprintf("%d-12", v.Year)), Runtime: v.Runtime, Preliminary: v.Preliminary, Source: v.Source, FetchedAt: v.FetchedAt})
	date := ""
	if ok {
		date = pub.Format(time.DateOnly)
	}
	status := "final"
	if v.Preliminary {
		status = "preliminary"
	}
	return ValorisationIndex{RevisionID: v.RevisionID, Series: string(v.Series), Period: fmt.Sprint(v.Year), Used: true, Value: v.Value.String(), Status: status, PublishedOn: date, Source: v.Source, PublicationSource: source}
}

// Beyond the evidenced publication calendar, a final observation is usable no
// earlier than the day its official status was retrieved. This is conservative:
// it never invents an earlier legal publication date from a monthly schedule.
func indexValuePublication(v indexation.IndexValue) (time.Time, string, bool) {
	if date, source, ok := finalIndexPublication(v.Month); ok {
		return date, source, true
	}
	if v.Runtime && !v.Preliminary && !v.FetchedAt.IsZero() {
		date, _ := time.Parse(time.DateOnly, v.FetchedAt.UTC().Format(time.DateOnly))
		return date, indexation.PeriodsURL(v.Source), true
	}
	return time.Time{}, "", false
}

func snapshotIndexPublication(snapshot indexation.Snapshot, series indexation.Series, month indexation.Month) (time.Time, string, bool) {
	if v, ok, err := snapshot.Data.Lookup(series, month); err == nil && ok {
		return indexValuePublication(v)
	}
	return finalIndexPublication(month)
}
