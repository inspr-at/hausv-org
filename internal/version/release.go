package version

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const Scheme = "inspr-calendar-v2"
const FirstCalendarVersion = "260914170935.0.0"
const LastLegacyVersion = "1.11.0"
const ReleaseSequence = 14
const FirstCalendarReleaseSequence = 1

// Channel is a separate artifact dimension, never a canonical version suffix.
var Channel = "production"

type ReleaseIdentity struct {
	VersionScheme   string `json:"version_scheme"`
	Version         string `json:"version"`
	ReleaseChannel  string `json:"release_channel"`
	ReleaseSequence uint64 `json:"release_sequence"`
	Commit          string `json:"commit"`
}

func Identity() ReleaseIdentity {
	scheme := Scheme
	if Version == "dev" || Version == "" {
		scheme = "development"
	}
	return ReleaseIdentity{scheme, Version, Channel, ReleaseSequence, Commit}
}
func IdentityJSON() string { b, _ := json.Marshal(Identity()); return string(b) }

var calendarPattern = regexp.MustCompile(`^[1-9][0-9]{11}\.0\.0$`)

func ParseCalendar(value string) (time.Time, error) {
	if !calendarPattern.MatchString(value) {
		return time.Time{}, fmt.Errorf("invalid calendar version")
	}
	stamp := value[:12]
	date, err := time.Parse("20060102150405", "20"+stamp)
	if err != nil || date.Year() < 2010 || date.Year() > 2099 || date.UTC().Format("060102150405") != stamp {
		return time.Time{}, fmt.Errorf("invalid calendar date")
	}
	return date, nil
}
func ValidateIdentity(r ReleaseIdentity) error {
	if r.ReleaseChannel == "" {
		return fmt.Errorf("release channel required")
	}
	switch r.VersionScheme {
	case Scheme:
		_, err := ParseCalendar(r.Version)
		return err
	case "legacy":
		if !regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`).MatchString(r.Version) {
			return fmt.Errorf("invalid legacy version")
		}
		return nil
	default:
		return fmt.Errorf("unsupported version scheme")
	}
}

// Cross-era comparison is permitted only across the explicit adoption anchor.
func Compare(a, b ReleaseIdentity) (int, error) {
	if err := ValidateIdentity(a); err != nil {
		return 0, err
	}
	if err := ValidateIdentity(b); err != nil {
		return 0, err
	}
	if a.ReleaseChannel != b.ReleaseChannel {
		return 0, fmt.Errorf("different channels")
	}
	if a.VersionScheme == Scheme && b.VersionScheme == Scheme {
		return strings.Compare(a.Version, b.Version), nil
	}
	if a.VersionScheme == "legacy" && a.Version == LastLegacyVersion && b.VersionScheme == Scheme && b.ReleaseSequence >= FirstCalendarReleaseSequence {
		return -1, nil
	}
	if b.VersionScheme == "legacy" && b.Version == LastLegacyVersion && a.VersionScheme == Scheme && a.ReleaseSequence >= FirstCalendarReleaseSequence {
		return 1, nil
	}
	if a.VersionScheme == b.VersionScheme && a.Version == b.Version {
		return 0, nil
	}
	return 0, fmt.Errorf("release ordering not established")
}
func Reserve(now time.Time, previous string) (string, error) {
	candidate := now.UTC().Format("060102150405") + ".0.0"
	if _, err := ParseCalendar(candidate); err != nil {
		return "", err
	}
	if previous != "" {
		if _, err := ParseCalendar(previous); err != nil {
			return "", err
		}
		if candidate <= previous {
			return "", fmt.Errorf("reservation must use a later UTC second")
		}
	}
	return candidate, nil
}

// Historical notes remain explicitly legacy; new entries carry Scheme.
func NoteScheme(note Note) string {
	if note.Scheme == "" {
		return "legacy"
	}
	return note.Scheme
}
