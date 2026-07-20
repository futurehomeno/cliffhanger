package adapter

import "testing"

func TestTopologyChecksum_NilReportDoesNotPanic(t *testing.T) {
	t.Parallel()

	sum, err := topologyChecksum(nil)
	if err != nil || sum != 0 {
		t.Fatalf("nil report: got (%d, %v), want (0, nil)", sum, err)
	}
}
