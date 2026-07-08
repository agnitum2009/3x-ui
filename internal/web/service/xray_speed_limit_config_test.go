package service

import (
	"encoding/json"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func emittedInboundSettings(t *testing.T, tag string) map[string]any {
	t.Helper()
	svc := &XrayService{}
	cfg, err := svc.GetXrayConfig()
	if err != nil {
		t.Fatalf("GetXrayConfig: %v", err)
	}
	for i := range cfg.InboundConfigs {
		ic := cfg.InboundConfigs[i]
		if ic.Tag != tag {
			continue
		}
		var s map[string]any
		if err := json.Unmarshal([]byte(ic.Settings), &s); err != nil {
			t.Fatalf("unmarshal emitted settings: %v", err)
		}
		return s
	}
	t.Fatalf("inbound %q not found in generated config", tag)
	return nil
}

func TestGetXrayConfigEmitsSpeedAndDeviceLimit(t *testing.T) {
	setupSettingTestDB(t)
	in := &model.Inbound{
		Tag:      "vless-speed",
		Enable:   true,
		Port:     42345,
		Protocol: model.VLESS,
		Settings: clientsSettings(t, []model.Client{{
			ID:         "11111111-1111-1111-1111-111111111111",
			Email:      "speed@vless.test",
			Enable:     true,
			LimitIP:    3,
			SpeedLimit: 1048576,
		}}),
	}
	if err := database.GetDB().Create(in).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	if err := (&ClientService{}).SyncInbound(nil, in.Id, []model.Client{{
		ID:         "11111111-1111-1111-1111-111111111111",
		Email:      "speed@vless.test",
		Enable:     true,
		LimitIP:    3,
		SpeedLimit: 1048576,
	}}); err != nil {
		t.Fatalf("SyncInbound: %v", err)
	}

	settings := emittedInboundSettings(t, "vless-speed")
	rawClients, ok := settings["clients"].([]any)
	if !ok || len(rawClients) != 1 {
		t.Fatalf("settings.clients = %T %v, want one client", settings["clients"], settings["clients"])
	}
	client, ok := rawClients[0].(map[string]any)
	if !ok {
		t.Fatalf("client is not an object: %T", rawClients[0])
	}
	if client["speedLimit"] != float64(1048576) {
		t.Fatalf("speedLimit = %v, want 1048576", client["speedLimit"])
	}
	if client["deviceLimit"] != float64(3) {
		t.Fatalf("deviceLimit = %v, want 3", client["deviceLimit"])
	}
}
