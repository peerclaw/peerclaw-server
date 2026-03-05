package mcp

import (
	"encoding/json"
	"testing"

	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
)

func TestInitializeParamsJSON(t *testing.T) {
	params := InitializeParams{
		ProtocolVersion: "2025-11-25",
		Capabilities: ClientCaps{
			Roots: &RootsCap{ListChanged: true},
		},
		ClientInfo: ImplementInfo{Name: "test-client", Version: "1.0"},
	}
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	var decoded InitializeParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ProtocolVersion != "2025-11-25" {
		t.Errorf("ProtocolVersion = %q", decoded.ProtocolVersion)
	}
	if decoded.Capabilities.Roots == nil || !decoded.Capabilities.Roots.ListChanged {
		t.Error("Roots.ListChanged should be true")
	}
}

func TestInitializeResultJSON(t *testing.T) {
	result := InitializeResult{
		ProtocolVersion: "2025-11-25",
		Capabilities: ServerCaps{
			Tools:     &ToolsCap{ListChanged: true},
			Resources: &ResourcesCap{Subscribe: true},
		},
		ServerInfo: ImplementInfo{Name: "test-server", Version: "1.0"},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded InitializeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Capabilities.Tools == nil {
		t.Fatal("Tools should not be nil")
	}
}

func TestToolDefJSON(t *testing.T) {
	tool := ToolDef{
		Name:        "search",
		Description: "Search the web",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
	}
	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ToolDef
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "search" {
		t.Errorf("Name = %q", decoded.Name)
	}
}

func TestToolCallResultJSON(t *testing.T) {
	result := ToolCallResult{
		Content: []Content{
			{Type: "text", Text: "Hello world"},
		},
		IsError: false,
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ToolCallResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Content) != 1 {
		t.Fatalf("Content len = %d", len(decoded.Content))
	}
	if decoded.Content[0].Text != "Hello world" {
		t.Errorf("Text = %q", decoded.Content[0].Text)
	}
}

func TestEnvelopeToToolCall_WithMetadata(t *testing.T) {
	env := envelope.New("a", "b", protocol.ProtocolMCP, []byte(`{"key":"value"}`))
	env.Metadata["mcp.tool_name"] = "search"

	tc, err := EnvelopeToToolCall(env)
	if err != nil {
		t.Fatal(err)
	}
	if tc.Name != "search" {
		t.Errorf("Name = %q", tc.Name)
	}
}

func TestEnvelopeToToolCall_FromPayload(t *testing.T) {
	payload, _ := json.Marshal(ToolCallParams{Name: "analyze", Arguments: json.RawMessage(`{"text":"hello"}`)})
	env := envelope.New("a", "b", protocol.ProtocolMCP, payload)

	tc, err := EnvelopeToToolCall(env)
	if err != nil {
		t.Fatal(err)
	}
	if tc.Name != "analyze" {
		t.Errorf("Name = %q", tc.Name)
	}
}

func TestToolResultToEnvelope(t *testing.T) {
	result := &ToolCallResult{
		Content: []Content{{Type: "text", Text: "result"}},
	}
	env := ToolResultToEnvelope(result, "server", "client")
	if env.Protocol != protocol.ProtocolMCP {
		t.Errorf("Protocol = %q", env.Protocol)
	}
	if env.MessageType != envelope.MessageTypeResponse {
		t.Errorf("MessageType = %q", env.MessageType)
	}
}

func TestToolResultToEnvelope_Error(t *testing.T) {
	result := &ToolCallResult{
		Content: []Content{{Type: "text", Text: "error"}},
		IsError: true,
	}
	env := ToolResultToEnvelope(result, "server", "client")
	if env.MessageType != envelope.MessageTypeError {
		t.Errorf("MessageType = %q", env.MessageType)
	}
}

func TestResourceJSON(t *testing.T) {
	r := Resource{
		URI:         "file:///test.txt",
		Name:        "test",
		Description: "A test resource",
		MimeType:    "text/plain",
	}
	data, _ := json.Marshal(r)
	var decoded Resource
	json.Unmarshal(data, &decoded)
	if decoded.URI != "file:///test.txt" {
		t.Errorf("URI = %q", decoded.URI)
	}
}

func TestPromptJSON(t *testing.T) {
	p := Prompt{
		Name: "greet",
		Arguments: []PromptArgument{
			{Name: "name", Required: true},
		},
	}
	data, _ := json.Marshal(p)
	var decoded Prompt
	json.Unmarshal(data, &decoded)
	if decoded.Name != "greet" {
		t.Errorf("Name = %q", decoded.Name)
	}
	if len(decoded.Arguments) != 1 || !decoded.Arguments[0].Required {
		t.Errorf("Arguments = %+v", decoded.Arguments)
	}
}
