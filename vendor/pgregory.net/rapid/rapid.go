package rapid

import (
	"fmt"
	"math/rand"
	"testing"
)

const defaultChecks = 96

type T struct {
	tb    *testing.T
	rand  *rand.Rand
	check int
}

type Generator[V any] struct {
	draw func(*T) V
}

func Check(tb *testing.T, property func(*T)) {
	tb.Helper()
	for i := 0; i < defaultChecks; i++ {
		rt := &T{
			tb:    tb,
			rand:  rand.New(rand.NewSource(int64(0x525047) + int64(i)*7919)),
			check: i,
		}
		property(rt)
	}
}

func Custom[V any](draw func(*T) V) *Generator[V] {
	return &Generator[V]{draw: draw}
}

func Bool() *Generator[bool] {
	return Custom(func(t *T) bool {
		return t.rand.Intn(2) == 0
	})
}

func IntRange(min int, max int) *Generator[int] {
	return Custom(func(t *T) int {
		if max < min {
			t.Fatalf("invalid IntRange(%d, %d)", min, max)
		}
		return min + t.rand.Intn(max-min+1)
	})
}

func SampledFrom[V any](values []V) *Generator[V] {
	return Custom(func(t *T) V {
		if len(values) == 0 {
			t.Fatalf("SampledFrom requires at least one value")
		}
		return values[t.rand.Intn(len(values))]
	})
}

func SliceOfN[V any](gen *Generator[V], min int, max int) *Generator[[]V] {
	return Custom(func(t *T) []V {
		n := IntRange(min, max).Draw(t, "len")
		out := make([]V, n)
		for i := range out {
			out[i] = gen.Draw(t, fmt.Sprintf("elem_%d", i))
		}
		return out
	})
}

func (g *Generator[V]) Draw(t *T, _ string) V {
	if g == nil || g.draw == nil {
		t.Fatalf("nil rapid generator")
	}
	return g.draw(t)
}

func (t *T) Helper() {
	t.tb.Helper()
}

func (t *T) Fatalf(format string, args ...any) {
	t.tb.Helper()
	t.tb.Fatalf("rapid check %d: "+format, append([]any{t.check}, args...)...)
}

func (t *T) Errorf(format string, args ...any) {
	t.tb.Helper()
	t.tb.Errorf("rapid check %d: "+format, append([]any{t.check}, args...)...)
}

func (t *T) Logf(format string, args ...any) {
	t.tb.Helper()
	t.tb.Logf("rapid check %d: "+format, append([]any{t.check}, args...)...)
}
