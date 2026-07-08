package model

import "testing"

func TestClientSpeedLimitRoundTrip(t *testing.T) {
	c := &Client{
		Email:      "speed@example.test",
		Enable:     true,
		LimitIP:    2,
		SpeedLimit: 1048576,
	}

	rec := c.ToRecord()
	if rec.SpeedLimit != c.SpeedLimit {
		t.Fatalf("ToRecord SpeedLimit = %d, want %d", rec.SpeedLimit, c.SpeedLimit)
	}

	got := rec.ToClient()
	if got.SpeedLimit != c.SpeedLimit {
		t.Fatalf("ToClient SpeedLimit = %d, want %d", got.SpeedLimit, c.SpeedLimit)
	}
}

func TestMergeClientRecordSpeedLimitPreservesNonZero(t *testing.T) {
	existing := &ClientRecord{Email: "speed@example.test", SpeedLimit: 1048576, UpdatedAt: 100}
	incomingEmpty := &ClientRecord{Email: "speed@example.test", UpdatedAt: 200}

	MergeClientRecord(existing, incomingEmpty)
	if existing.SpeedLimit != 1048576 {
		t.Fatalf("empty incoming wiped SpeedLimit: %d", existing.SpeedLimit)
	}

	incomingHigher := &ClientRecord{Email: "speed@example.test", SpeedLimit: 2097152, UpdatedAt: 300}
	conflicts := MergeClientRecord(existing, incomingHigher)
	if existing.SpeedLimit != 2097152 {
		t.Fatalf("higher incoming SpeedLimit not applied: %d", existing.SpeedLimit)
	}
	if len(conflicts) == 0 || conflicts[len(conflicts)-1].Field != "speedLimit" {
		t.Fatalf("speedLimit conflict not recorded: %#v", conflicts)
	}
}
