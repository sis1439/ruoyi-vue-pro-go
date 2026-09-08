package alipay

import "testing"

func TestChannelAmountExactFen(t *testing.T) {
	for raw, want := range map[string]int{"0.01": 1, "1": 100, "1.2": 120, "30000000.01": 3000000001} {
		got, err := parseAmount(raw)
		if err != nil || got != want {
			t.Fatalf("%s: %d %v", raw, got, err)
		}
		if back, err := parseAmount(formatAmount(want)); err != nil || back != want {
			t.Fatal("amount roundtrip")
		}
	}
	for _, raw := range []string{"", "0", "0.00", "1.001", "-1", "+1", "1e2", "NaN", "999999999999999999999999"} {
		if _, err := parseAmount(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}
