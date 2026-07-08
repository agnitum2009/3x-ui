package service

import (
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestUpdate_PersistsRecordEnable_True(t *testing.T) {
	setupBulkDB(t)
	svc := &ClientService{}
	inboundSvc := &InboundService{}

	email := "u-true@x"
	c := model.Client{Email: email, ID: "11111111-1111-1111-1111-111111111111", SubID: email, Enable: false}
	ib := mkInbound(t, 53001, model.VLESS, clientsSettings(t, []model.Client{c}))
	if err := svc.SyncInbound(nil, ib.Id, []model.Client{c}); err != nil {
		t.Fatalf("seed linkage: %v", err)
	}
	mkTraffic(t, ib.Id, email, 0, 0, 0, 0, false)

	rec, err := svc.GetRecordByEmail(nil, email)
	if err != nil {
		t.Fatalf("GetRecordByEmail: %v", err)
	}
	updated := rec.ToClient()
	updated.Enable = true
	if _, err := svc.Update(inboundSvc, rec.Id, *updated); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got := recordEnableOf(t, svc, email); !got {
		t.Fatalf("%s: client_records.enable = false, want true", email)
	}
	if got := trafficOf(t, email).Enable; !got {
		t.Fatalf("%s: client_traffics.enable = false, want true", email)
	}
	if got := jsonClientEnable(t, inboundSvc, ib.Id, email); !got {
		t.Fatalf("%s: inbound JSON enable = false, want true", email)
	}
}

func TestUpdate_PersistsRecordEnable_False(t *testing.T) {
	setupBulkDB(t)
	svc := &ClientService{}
	inboundSvc := &InboundService{}

	email := "u-false@x"
	c := model.Client{Email: email, ID: "11111111-1111-1111-1111-111111111111", SubID: email, Enable: true}
	ib := mkInbound(t, 53002, model.VLESS, clientsSettings(t, []model.Client{c}))
	if err := svc.SyncInbound(nil, ib.Id, []model.Client{c}); err != nil {
		t.Fatalf("seed linkage: %v", err)
	}
	mkTraffic(t, ib.Id, email, 0, 0, 0, 0, true)

	rec, err := svc.GetRecordByEmail(nil, email)
	if err != nil {
		t.Fatalf("GetRecordByEmail: %v", err)
	}
	updated := rec.ToClient()
	updated.Enable = false
	if _, err := svc.Update(inboundSvc, rec.Id, *updated); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got := recordEnableOf(t, svc, email); got {
		t.Fatalf("%s: client_records.enable = true, want false", email)
	}
}

func TestUpdate_PersistsRecordEnable_NoInbound(t *testing.T) {
	setupBulkDB(t)
	svc := &ClientService{}
	inboundSvc := &InboundService{}

	email := "u-noib@x"
	rec := &model.ClientRecord{
		Email:  email,
		UUID:   "11111111-1111-1111-1111-111111111111",
		SubID:  email,
		Enable: false,
	}
	if err := database.GetDB().Create(rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}
	forceRecordDisabled(t, svc, email)

	updated := rec.ToClient()
	updated.Enable = true
	if _, err := svc.Update(inboundSvc, rec.Id, *updated); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got := recordEnableOf(t, svc, email); !got {
		t.Fatalf("%s: client_records.enable = false, want true (no-inbound persistence gap)", email)
	}
}

func TestResetTrafficByEmail_LeavesRecordEnableTrue(t *testing.T) {
	setupBulkDB(t)
	svc := &ClientService{}
	inboundSvc := &InboundService{}

	email := "r-attached@x"
	c := model.Client{Email: email, ID: "11111111-1111-1111-1111-111111111111", SubID: email, Enable: false}
	ib := mkInbound(t, 53003, model.VLESS, clientsSettings(t, []model.Client{c}))
	if err := svc.SyncInbound(nil, ib.Id, []model.Client{c}); err != nil {
		t.Fatalf("seed linkage: %v", err)
	}
	mkTraffic(t, ib.Id, email, 10, 20, 0, 0, false)

	if _, err := svc.ResetTrafficByEmail(inboundSvc, email); err != nil {
		t.Fatalf("ResetTrafficByEmail: %v", err)
	}

	if got := recordEnableOf(t, svc, email); !got {
		t.Fatalf("%s: client_records.enable = false, want true", email)
	}
	tr := trafficOf(t, email)
	if !tr.Enable {
		t.Fatalf("%s: client_traffics.enable = false, want true", email)
	}
	if tr.Up != 0 || tr.Down != 0 {
		t.Fatalf("%s: expected up/down 0, got up=%d down=%d", email, tr.Up, tr.Down)
	}
}

func TestResetTrafficByEmail_NoInbound_LeavesRecordEnableTrue(t *testing.T) {
	setupBulkDB(t)
	svc := &ClientService{}
	inboundSvc := &InboundService{}

	email := "r-noib@x"
	rec := &model.ClientRecord{
		Email:  email,
		UUID:   "11111111-1111-1111-1111-111111111111",
		SubID:  email,
		Enable: false,
	}
	if err := database.GetDB().Create(rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}
	forceRecordDisabled(t, svc, email)

	if _, err := svc.ResetTrafficByEmail(inboundSvc, email); err != nil {
		t.Fatalf("ResetTrafficByEmail: %v", err)
	}

	if got := recordEnableOf(t, svc, email); !got {
		t.Fatalf("%s: client_records.enable = false, want true (no-inbound reset re-enable gap)", email)
	}
}

func TestUpdate_PersistsDirectionalSpeedLimits(t *testing.T) {
	setupBulkDB(t)
	svc := &ClientService{}
	inboundSvc := &InboundService{}

	email := "speed-update@x"
	c := model.Client{Email: email, ID: "11111111-1111-1111-1111-111111111111", SubID: email, Enable: true}
	ib := mkInbound(t, 53005, model.VLESS, clientsSettings(t, []model.Client{c}))
	if err := svc.SyncInbound(nil, ib.Id, []model.Client{c}); err != nil {
		t.Fatalf("seed linkage: %v", err)
	}

	rec, err := svc.GetRecordByEmail(nil, email)
	if err != nil {
		t.Fatalf("GetRecordByEmail: %v", err)
	}
	updated := rec.ToClient()
	updated.UpSpeedLimit = 5 * 1024 * 1024
	updated.DownSpeedLimit = 10 * 1024 * 1024
	updated.SessionLimit = 128
	if _, err := svc.Update(inboundSvc, rec.Id, *updated); err != nil {
		t.Fatalf("Update set speed: %v", err)
	}
	assertSpeedLimits(t, svc, inboundSvc, ib.Id, email, updated.UpSpeedLimit, updated.DownSpeedLimit)
	assertSessionLimit(t, svc, inboundSvc, ib.Id, email, 128)

	updated.UpSpeedLimit = 0
	updated.DownSpeedLimit = 0
	updated.SpeedLimit = 0
	updated.SessionLimit = 0
	if _, err := svc.Update(inboundSvc, rec.Id, *updated); err != nil {
		t.Fatalf("Update clear speed: %v", err)
	}
	assertSpeedLimits(t, svc, inboundSvc, ib.Id, email, 0, 0)
	assertSessionLimit(t, svc, inboundSvc, ib.Id, email, 0)
}

func assertSessionLimit(t *testing.T, svc *ClientService, inboundSvc *InboundService, inboundId int, email string, want uint32) {
	t.Helper()
	rec, err := svc.GetRecordByEmail(nil, email)
	if err != nil {
		t.Fatalf("GetRecordByEmail(%q): %v", email, err)
	}
	if rec.SessionLimit != want {
		t.Fatalf("record sessionLimit=%d, want %d", rec.SessionLimit, want)
	}
	c := jsonClientByEmail(t, inboundSvc, inboundId, email)
	if c.SessionLimit != want {
		t.Fatalf("inbound JSON sessionLimit=%d, want %d", c.SessionLimit, want)
	}
}

func assertSpeedLimits(t *testing.T, svc *ClientService, inboundSvc *InboundService, inboundId int, email string, wantUp, wantDown uint64) {
	t.Helper()
	rec, err := svc.GetRecordByEmail(nil, email)
	if err != nil {
		t.Fatalf("GetRecordByEmail(%q): %v", email, err)
	}
	if rec.UpSpeedLimit != wantUp || rec.DownSpeedLimit != wantDown || rec.SpeedLimit != wantDown {
		t.Fatalf("record speed limits up=%d down=%d legacy=%d, want up=%d down=%d", rec.UpSpeedLimit, rec.DownSpeedLimit, rec.SpeedLimit, wantUp, wantDown)
	}
	c := jsonClientByEmail(t, inboundSvc, inboundId, email)
	if c.UpSpeedLimit != wantUp || c.DownSpeedLimit != wantDown || c.SpeedLimit != wantDown {
		t.Fatalf("inbound JSON speed limits up=%d down=%d legacy=%d, want up=%d down=%d", c.UpSpeedLimit, c.DownSpeedLimit, c.SpeedLimit, wantUp, wantDown)
	}
}

func jsonClientByEmail(t *testing.T, inboundSvc *InboundService, inboundId int, email string) model.Client {
	t.Helper()
	ib, err := inboundSvc.GetInbound(inboundId)
	if err != nil {
		t.Fatalf("GetInbound(%d): %v", inboundId, err)
	}
	clients, err := inboundSvc.GetClients(ib)
	if err != nil {
		t.Fatalf("GetClients(%d): %v", inboundId, err)
	}
	for _, c := range clients {
		if c.Email == email {
			return c
		}
	}
	t.Fatalf("client %q not found in inbound %d settings JSON", email, inboundId)
	return model.Client{}
}
