package model

import "testing"

func TestClientSpeedLimitRoundTrip(t *testing.T) {
	c := &Client{
		Email:          "speed@example.test",
		Enable:         true,
		LimitIP:        2,
		UpSpeedLimit:   524288,
		DownSpeedLimit: 1048576,
	}

	rec := c.ToRecord()
	if rec.UpSpeedLimit != c.UpSpeedLimit || rec.DownSpeedLimit != c.DownSpeedLimit || rec.SpeedLimit != c.DownSpeedLimit {
		t.Fatalf("ToRecord speed limits up=%d down=%d legacy=%d, want up=%d down=%d", rec.UpSpeedLimit, rec.DownSpeedLimit, rec.SpeedLimit, c.UpSpeedLimit, c.DownSpeedLimit)
	}

	got := rec.ToClient()
	if got.UpSpeedLimit != c.UpSpeedLimit || got.DownSpeedLimit != c.DownSpeedLimit || got.SpeedLimit != c.DownSpeedLimit {
		t.Fatalf("ToClient speed limits up=%d down=%d legacy=%d, want up=%d down=%d", got.UpSpeedLimit, got.DownSpeedLimit, got.SpeedLimit, c.UpSpeedLimit, c.DownSpeedLimit)
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

func TestMergeClientRecordDirectionalSpeedLimitPreservesNonZero(t *testing.T) {
	existing := &ClientRecord{Email: "speed@example.test", UpSpeedLimit: 524288, DownSpeedLimit: 1048576, SpeedLimit: 1048576, UpdatedAt: 100}
	incomingEmpty := &ClientRecord{Email: "speed@example.test", UpdatedAt: 200}

	MergeClientRecord(existing, incomingEmpty)
	if existing.UpSpeedLimit != 524288 || existing.DownSpeedLimit != 1048576 {
		t.Fatalf("empty incoming wiped speed limits: up=%d down=%d", existing.UpSpeedLimit, existing.DownSpeedLimit)
	}

	incomingHigher := &ClientRecord{Email: "speed@example.test", UpSpeedLimit: 1048576, DownSpeedLimit: 2097152, UpdatedAt: 300}
	conflicts := MergeClientRecord(existing, incomingHigher)
	if existing.UpSpeedLimit != 1048576 || existing.DownSpeedLimit != 2097152 || existing.SpeedLimit != 2097152 {
		t.Fatalf("higher incoming speed limits not applied: up=%d down=%d legacy=%d", existing.UpSpeedLimit, existing.DownSpeedLimit, existing.SpeedLimit)
	}
	if len(conflicts) < 2 || conflicts[len(conflicts)-2].Field != "upSpeedLimit" || conflicts[len(conflicts)-1].Field != "downSpeedLimit" {
		t.Fatalf("directional speed limit conflicts not recorded: %#v", conflicts)
	}
}
