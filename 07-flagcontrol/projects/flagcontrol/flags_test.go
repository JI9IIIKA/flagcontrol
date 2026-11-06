package flagcontrol

import (
	"fmt"
	"testing"
)

func TestStableRollout(t *testing.T) {
	enabled := 0
	for i := 0; i < 10000; i++ {
		user := fmt.Sprint(i)
		first := Enabled("tenant", "flag", "prod", user, true, 30, nil)
		if first != Enabled("tenant", "flag", "prod", user, true, 30, nil) {
			t.Fatal("unstable cohort")
		}
		if first {
			enabled++
		}
	}
	if enabled < 2700 || enabled > 3300 {
		t.Fatalf("unexpected distribution %d", enabled)
	}
}
func TestRolloutBoundaries(t *testing.T) {
	if Enabled("t", "k", "e", "u", false, 100, []string{"u"}) {
		t.Fatal("disabled flag allowed user")
	}
	if Enabled("t", "k", "e", "u", true, 0, nil) {
		t.Fatal("zero rollout enabled")
	}
	if !Enabled("t", "k", "e", "u", true, 0, []string{"u"}) {
		t.Fatal("allow list ignored")
	}
	if !Enabled("t", "k", "e", "u", true, 100, nil) {
		t.Fatal("full rollout disabled")
	}
}
