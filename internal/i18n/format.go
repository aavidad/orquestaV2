package i18n

import (
	"fmt"
	"math"
	"strings"
	"time"
	_ "time/tzdata"

	"golang.org/x/text/currency"
	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

const microsPerUnit = uint64(1_000_000)

func (catalog *Catalog) Plural(locale, key string, count int64) (string, error) {
	resolution, err := catalog.Resolve(locale)
	if err != nil {
		return "", err
	}
	value, err := catalog.message(resolution.Locale, key)
	if err != nil {
		return "", err
	}
	if value.kind != pluralMessage {
		return "", fmt.Errorf("%w: key=%s", ErrMessageKind, key)
	}
	form := cardinalForm(catalog.tags[resolution.Locale], count)
	template, ok := value.forms[form]
	if !ok {
		template = value.forms[plural.Other]
	}
	return message.NewPrinter(catalog.tags[resolution.Locale]).Sprintf(template, count), nil
}

func (catalog *Catalog) FormatNumber(locale string, value int64) (string, error) {
	resolution, err := catalog.Resolve(locale)
	if err != nil {
		return "", err
	}
	printer := message.NewPrinter(catalog.tags[resolution.Locale])
	return printer.Sprint(number.Decimal(value)), nil
}

func (catalog *Catalog) FormatCurrency(locale, isoCode string, micros int64) (string, error) {
	resolution, err := catalog.Resolve(locale)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(isoCode) != isoCode || len(isoCode) != 3 ||
		strings.ToUpper(isoCode) != isoCode {
		return "", fmt.Errorf("%w: %q", ErrCurrencyInvalid, isoCode)
	}
	unit, err := currency.ParseISO(isoCode)
	if err != nil || unit.String() != isoCode {
		return "", fmt.Errorf("%w: %q", ErrCurrencyInvalid, isoCode)
	}
	absolute := absoluteInt64(micros)
	whole := absolute / microsPerUnit
	fraction := absolute % microsPerUnit
	printer := message.NewPrinter(catalog.tags[resolution.Locale])
	formattedWhole := printer.Sprint(number.Decimal(whole))
	sign := ""
	if micros < 0 {
		sign = "-"
	}
	return fmt.Sprintf("%s %s%s%s%06d", unit.String(), sign, formattedWhole,
		decimalSeparator(resolution.Locale), fraction), nil
}

func (catalog *Catalog) FormatDate(locale string, value time.Time) (string, error) {
	resolution, err := catalog.Resolve(locale)
	if err != nil {
		return "", err
	}
	if resolution.Locale == "en" {
		return value.Format("01/02/2006"), nil
	}
	return value.Format("02/01/2006"), nil
}

func (catalog *Catalog) FormatTimeZone(locale string, value time.Time, zone string) (string, error) {
	resolution, err := catalog.Resolve(locale)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(zone) != zone || zone == "" || zone == "Local" {
		return "", fmt.Errorf("%w: %q", ErrTimeZoneInvalid, zone)
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrTimeZoneInvalid, zone)
	}
	local := value.In(location)
	if resolution.Locale == "en" {
		return local.Format("01/02/2006 03:04:05 PM -07:00 ") + zone, nil
	}
	return local.Format("02/01/2006 15:04:05 -07:00 ") + zone, nil
}

func cardinalForm(tag language.Tag, count int64) plural.Form {
	operand := absoluteInt64(count)
	if operand > uint64(^uint(0)>>1) {
		return plural.Other
	}
	return plural.Cardinal.MatchPlural(tag, int(operand), 0, 0, 0, 0)
}

func absoluteInt64(value int64) uint64 {
	if value >= 0 {
		return uint64(value)
	}
	if value == math.MinInt64 {
		return uint64(math.MaxInt64) + 1
	}
	return uint64(-value)
}

func decimalSeparator(locale string) string {
	if locale == "en" {
		return "."
	}
	return ","
}
