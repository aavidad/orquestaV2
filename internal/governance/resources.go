package governance

import (
	"math"
	"strings"
)

// Currency is a canonical uppercase three-letter monetary unit.
// Empty is valid only when a vector has no positive monetary amount.
type Currency string

// NewCurrency normalizes an ISO-style alphabetic code to its canonical form.
func NewCurrency(value string) (Currency, error) {
	currency := Currency(strings.ToUpper(value))
	if !validCurrency(currency) {
		return "", domainError(ErrorInvalidCurrency, "currency")
	}
	return currency, nil
}

func validCurrency(currency Currency) bool {
	if len(currency) != 3 {
		return false
	}
	for _, character := range currency {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}

// ResourceVector is the single integer resource vocabulary used by plans,
// envelopes, reservations, provider observations, and settlements.
type ResourceVector struct {
	Tokens       int64
	MoneyMicros  int64
	Currency     Currency
	ActiveTimeNS int64
	ProcessSlots int64
	DiskBytes    int64
}

// ValidateResourceVector rejects negative values and noncanonical money.
func ValidateResourceVector(vector ResourceVector) error {
	values := []struct {
		field string
		value int64
	}{
		{"tokens", vector.Tokens}, {"money_micros", vector.MoneyMicros},
		{"active_time_ns", vector.ActiveTimeNS}, {"process_slots", vector.ProcessSlots},
		{"disk_bytes", vector.DiskBytes},
	}
	for _, value := range values {
		if value.value < 0 {
			return domainError(ErrorInvalidArgument, value.field)
		}
	}
	if vector.Currency != "" && !validCurrency(vector.Currency) {
		return domainError(ErrorInvalidCurrency, "currency")
	}
	if vector.MoneyMicros > 0 && vector.Currency == "" {
		return domainError(ErrorInvalidCurrency, "currency")
	}
	return nil
}

// Add sums two nonnegative vectors without wrapping. Currency is preserved
// from either side and must agree when present on both.
func Add(left, right ResourceVector) (ResourceVector, error) {
	if err := ValidateResourceVector(left); err != nil {
		return ResourceVector{}, err
	}
	if err := ValidateResourceVector(right); err != nil {
		return ResourceVector{}, err
	}
	currency, err := mergeCurrency(left.Currency, right.Currency)
	if err != nil {
		return ResourceVector{}, err
	}
	values := [5]int64{}
	for index, pair := range [][2]int64{
		{left.Tokens, right.Tokens}, {left.MoneyMicros, right.MoneyMicros},
		{left.ActiveTimeNS, right.ActiveTimeNS}, {left.ProcessSlots, right.ProcessSlots},
		{left.DiskBytes, right.DiskBytes},
	} {
		if pair[0] > math.MaxInt64-pair[1] {
			return ResourceVector{}, domainError(ErrorOverflow, resourceField(index))
		}
		values[index] = pair[0] + pair[1]
	}
	return ResourceVector{
		Tokens: values[0], MoneyMicros: values[1], Currency: currency,
		ActiveTimeNS: values[2], ProcessSlots: values[3], DiskBytes: values[4],
	}, nil
}

// Fits reports whether requested is within every finite limit dimension.
func Fits(limit, requested ResourceVector) (bool, error) {
	if err := ValidateResourceVector(limit); err != nil {
		return false, err
	}
	if err := ValidateResourceVector(requested); err != nil {
		return false, err
	}
	if _, err := mergeCurrency(limit.Currency, requested.Currency); err != nil {
		return false, err
	}
	return requested.Tokens <= limit.Tokens && requested.MoneyMicros <= limit.MoneyMicros &&
		requested.ActiveTimeNS <= limit.ActiveTimeNS && requested.ProcessSlots <= limit.ProcessSlots &&
		requested.DiskBytes <= limit.DiskBytes, nil
}

func mergeCurrency(left, right Currency) (Currency, error) {
	if left != "" && right != "" && left != right {
		return "", domainError(ErrorCurrencyConflict, "currency")
	}
	if left != "" {
		return left, nil
	}
	return right, nil
}

func resourceField(index int) string {
	return [...]string{"tokens", "money_micros", "active_time_ns", "process_slots", "disk_bytes"}[index]
}
