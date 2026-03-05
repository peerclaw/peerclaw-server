package jsonrpc

import (
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	req, err := NewRequest("test.method", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if req.JSONRPC != Version {
		t.Errorf("JSONRPC = %q, want %q", req.JSONRPC, Version)
	}
	if req.Method != "test.method" {
		t.Errorf("Method = %q, want %q", req.Method, "test.method")
	}
	if req.ID == nil {
		t.Error("ID should not be nil")
	}

	// Roundtrip
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Method != "test.method" {
		t.Errorf("roundtrip Method = %q, want %q", decoded.Method, "test.method")
	}
}

func TestNewRequestWithID(t *testing.T) {
	req, err := NewRequestWithID("abc-123", "test.method", nil)
	if err != nil {
		t.Fatalf("NewRequestWithID: %v", err)
	}
	if req.ID != "abc-123" {
		t.Errorf("ID = %v, want %q", req.ID, "abc-123")
	}
	if req.Params != nil {
		t.Errorf("Params should be nil for nil input")
	}
}

func TestNewResponse(t *testing.T) {
	resp, err := NewResponse(1, map[string]string{"status": "ok"})
	if err != nil {
		t.Fatalf("NewResponse: %v", err)
	}
	if resp.Error != nil {
		t.Error("Error should be nil for success response")
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Error != nil {
		t.Error("roundtrip Error should be nil")
	}
}

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse(1, CodeMethodNotFound, "method not found")
	if resp.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if resp.Error.Code != CodeMethodNotFound {
		t.Errorf("Error.Code = %d, want %d", resp.Error.Code, CodeMethodNotFound)
	}
	if resp.Error.Error() != "jsonrpc error -32601: method not found" {
		t.Errorf("Error.Error() = %q", resp.Error.Error())
	}
}

func TestNewNotification(t *testing.T) {
	notif, err := NewNotification("notifications/initialized", nil)
	if err != nil {
		t.Fatalf("NewNotification: %v", err)
	}
	if notif.JSONRPC != Version {
		t.Errorf("JSONRPC = %q, want %q", notif.JSONRPC, Version)
	}
	if notif.Method != "notifications/initialized" {
		t.Errorf("Method = %q", notif.Method)
	}
}

func TestParseMessage_Request(t *testing.T) {
	data := `{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"text":"hello"}}`
	msg, err := ParseMessage([]byte(data))
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if msg.Kind != KindRequest {
		t.Errorf("Kind = %d, want KindRequest", msg.Kind)
	}
	if msg.Request == nil {
		t.Fatal("Request should not be nil")
	}
	if msg.Request.Method != "SendMessage" {
		t.Errorf("Method = %q, want %q", msg.Request.Method, "SendMessage")
	}
}

func TestParseMessage_Response(t *testing.T) {
	data := `{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`
	msg, err := ParseMessage([]byte(data))
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if msg.Kind != KindResponse {
		t.Errorf("Kind = %d, want KindResponse", msg.Kind)
	}
	if msg.Response == nil {
		t.Fatal("Response should not be nil")
	}
}

func TestParseMessage_ErrorResponse(t *testing.T) {
	data := `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found"}}`
	msg, err := ParseMessage([]byte(data))
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if msg.Kind != KindResponse {
		t.Errorf("Kind = %d, want KindResponse", msg.Kind)
	}
	if msg.Response.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if msg.Response.Error.Code != CodeMethodNotFound {
		t.Errorf("Error.Code = %d", msg.Response.Error.Code)
	}
}

func TestParseMessage_Notification(t *testing.T) {
	data := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	msg, err := ParseMessage([]byte(data))
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if msg.Kind != KindNotification {
		t.Errorf("Kind = %d, want KindNotification", msg.Kind)
	}
	if msg.Notification == nil {
		t.Fatal("Notification should not be nil")
	}
	if msg.Notification.Method != "notifications/initialized" {
		t.Errorf("Method = %q", msg.Notification.Method)
	}
}

func TestParseMessage_InvalidJSON(t *testing.T) {
	_, err := ParseMessage([]byte("{invalid}"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseMessage_Unrecognized(t *testing.T) {
	_, err := ParseMessage([]byte(`{"jsonrpc":"2.0","id":1}`))
	if err == nil {
		t.Error("expected error for unrecognized message")
	}
}

func TestRequestRoundtrip(t *testing.T) {
	type params struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	req, err := NewRequest("test", params{Name: "alice", Age: 30})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseMessage(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Kind != KindRequest {
		t.Fatalf("expected request, got %d", parsed.Kind)
	}
	var p params
	if err := json.Unmarshal(parsed.Request.Params, &p); err != nil {
		t.Fatal(err)
	}
	if p.Name != "alice" || p.Age != 30 {
		t.Errorf("params = %+v", p)
	}
}
