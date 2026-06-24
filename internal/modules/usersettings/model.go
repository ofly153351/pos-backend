package usersettings

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// CardSettings is the per-user POS product-card display preference.
// Mirrors the frontend CardSettings shape. Implements driver.Valuer + sql.Scanner
// so it round-trips through the users.card_settings JSONB column.
type CardSettings struct {
	NamePos        string `json:"namePos"` // "bottom" | "top"
	Fit            string `json:"fit"`     // "cover" | "contain"
	Aspect         string `json:"aspect"`  // "1/1" | "4/3" | "3/4"
	Lines          int    `json:"lines"`   // 1 | 2 | 3
	Size           string `json:"size"`    // "sm" | "md" | "lg"
	ShowStock      bool   `json:"showStock"`
	ShowPromoBadge bool   `json:"showPromoBadge"`
}

// Value serialises to JSON for the JSONB column.
func (c CardSettings) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// UnmarshalJSON defaults ShowPromoBadge to true when the key is ABSENT, so rows
// persisted before this field existed (and older clients) keep the badge visible —
// while still honouring an explicit "showPromoBadge": false. Other fields keep their
// zero value when absent (normalize() then backfills the string/int ones).
func (c *CardSettings) UnmarshalJSON(data []byte) error {
	type alias CardSettings
	tmp := alias{ShowPromoBadge: true}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*c = CardSettings(tmp)
	return nil
}

// Scan deserialises a JSONB value ([]byte or string) into the struct.
func (c *CardSettings) Scan(value any) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			return nil
		}
		return json.Unmarshal(v, c)
	case string:
		if v == "" {
			return nil
		}
		return json.Unmarshal([]byte(v), c)
	default:
		return fmt.Errorf("usersettings: cannot scan %T into CardSettings", value)
	}
}

// DefaultCardSettings matches the frontend DEFAULT_CARD_SETTINGS.
func DefaultCardSettings() CardSettings {
	return CardSettings{
		NamePos:        "bottom",
		Fit:            "cover",
		Aspect:         "1/1",
		Lines:          2,
		Size:           "md",
		ShowStock:      true,
		ShowPromoBadge: true,
	}
}

// normalize clamps each field to an allowed value, falling back to the default.
func (c CardSettings) normalize() CardSettings {
	d := DefaultCardSettings()
	out := c

	if out.NamePos != "bottom" && out.NamePos != "top" {
		out.NamePos = d.NamePos
	}
	if out.Fit != "cover" && out.Fit != "contain" {
		out.Fit = d.Fit
	}
	switch out.Aspect {
	case "1/1", "4/3", "3/4":
	default:
		out.Aspect = d.Aspect
	}
	if out.Lines < 1 || out.Lines > 3 {
		out.Lines = d.Lines
	}
	switch out.Size {
	case "sm", "md", "lg":
	default:
		out.Size = d.Size
	}
	return out
}
