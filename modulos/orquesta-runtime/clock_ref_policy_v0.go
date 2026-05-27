package orquestaruntime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const (
	RefGenerationEntropyUnavailableV0 = "ref_entropy_unavailable"
	RefGenerationCollisionDetectedV0  = "ref_collision_detected"
)

type ClockV0 interface {
	Now() time.Time
}

type ClockFuncV0 func() time.Time

func (fn ClockFuncV0) Now() time.Time {
	if fn == nil {
		return time.Time{}
	}
	return fn()
}

type SystemClockV0 struct{}

func (SystemClockV0) Now() time.Time {
	return time.Now().UTC()
}

type RefGenerationIssueV0 struct {
	Code    string
	Message string
}

type RefGeneratorV0 struct {
	clock    ClockV0
	entropy  io.Reader
	mu       sync.Mutex
	counters map[string]uint64
	seen     map[string]bool
}

func NewSystemRefGeneratorV0() *RefGeneratorV0 {
	return NewRefGeneratorV0(SystemClockV0{}, rand.Reader)
}

func NewRefGeneratorV0(clock ClockV0, entropy io.Reader) *RefGeneratorV0 {
	if clock == nil {
		clock = SystemClockV0{}
	}
	if entropy == nil {
		entropy = rand.Reader
	}
	return &RefGeneratorV0{
		clock:    clock,
		entropy:  entropy,
		counters: map[string]uint64{},
		seen:     map[string]bool{},
	}
}

func NowUTCV0(clock ClockV0) time.Time {
	if clock == nil {
		clock = SystemClockV0{}
	}
	now := clock.Now().UTC()
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now
}

func FormatRFC3339UTCV0(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func NowRFC3339UTCV0(clock ClockV0) string {
	return FormatRFC3339UTCV0(NowUTCV0(clock))
}

func (generator *RefGeneratorV0) NextRefV0(
	prefix string,
	scopeParts ...string,
) (string, []RefGenerationIssueV0) {
	if generator == nil {
		generator = NewSystemRefGeneratorV0()
	}
	prefix = normalizeRefPrefixV0(prefix)
	scope := normalizeRefScopeV0(scopeParts...)
	key := prefix + scope

	generator.mu.Lock()
	defer generator.mu.Unlock()

	token, issues := generator.refTokenLockedV0(key)
	ref := prefix + scope + "-" + token
	if !generator.seen[ref] {
		generator.seen[ref] = true
		return ref, issues
	}
	issues = append(issues, RefGenerationIssueV0{
		Code:    RefGenerationCollisionDetectedV0,
		Message: "ref collision detected; counter suffix applied",
	})
	generator.counters[key]++
	ref = fmt.Sprintf("%s%s-%s-%06d", prefix, scope, token, generator.counters[key])
	generator.seen[ref] = true
	return ref, issues
}

func (generator *RefGeneratorV0) refTokenLockedV0(key string) (string, []RefGenerationIssueV0) {
	var raw [8]byte
	if _, err := io.ReadFull(generator.entropy, raw[:]); err == nil {
		return hex.EncodeToString(raw[:]), nil
	}
	generator.counters[key]++
	now := NowUTCV0(generator.clock)
	stamp := now.Format("20060102T150405") + fmt.Sprintf("%09dZ", now.Nanosecond())
	return fmt.Sprintf("degraded-%s-%06d", stamp, generator.counters[key]), []RefGenerationIssueV0{{
		Code:    RefGenerationEntropyUnavailableV0,
		Message: "entropy unavailable; degraded ref includes utc time and monotonic counter",
	}}
}

func normalizeRefPrefixV0(value string) string {
	value = normalizeRefTokenV0(value)
	if value == "" {
		value = "ref"
	}
	if !strings.HasSuffix(value, "-") {
		value += "-"
	}
	return value
}

func normalizeRefScopeV0(parts ...string) string {
	joined := strings.Join(parts, "-")
	scope := normalizeRefTokenV0(joined)
	if scope == "" {
		return "scope"
	}
	if len(scope) > 64 {
		return strings.Trim(scope[:64], "-")
	}
	return scope
}

func normalizeRefTokenV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if valid {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
