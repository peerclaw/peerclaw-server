package acp

import (
	"encoding/json"
	"testing"

	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
)

func TestRunStatusValues(t *testing.T) {
	statuses := []RunStatus{
		RunStatusCreated, RunStatusInProgress, RunStatusAwaiting,
		RunStatusCompleted, RunStatusFailed, RunStatusCancelling, RunStatusCancelled,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty run status")
		}
	}
}

func TestRunJSON(t *testing.T) {
	run := &Run{
		AgentName: "echo",
		RunID:     "run-1",
		SessionID: "sess-1",
		Status:    RunStatusCompleted,
		Input: []Message{
			{Role: "user", Parts: []MessagePart{{ContentType: "text/plain", Content: "hello"}}},
		},
		Output: []Message{
			{Role: "agent", Parts: []MessagePart{{ContentType: "text/plain", Content: "world"}}},
		},
		CreatedAt: "2025-01-01T00:00:00Z",
		UpdatedAt: "2025-01-01T00:00:00Z",
	}

	data, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Run
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.RunID != "run-1" {
		t.Errorf("RunID = %q", decoded.RunID)
	}
	if decoded.Status != RunStatusCompleted {
		t.Errorf("Status = %q", decoded.Status)
	}
}

func TestEnvelopeToCreateRun(t *testing.T) {
	env := envelope.New("alice", "echo-agent", protocol.ProtocolACP, []byte("do something"))
	env.Metadata["acp.agent_name"] = "echo"
	env.Metadata["acp.session_id"] = "sess-1"

	req := EnvelopeToCreateRun(env)
	if req.AgentName != "echo" {
		t.Errorf("AgentName = %q", req.AgentName)
	}
	if req.SessionID != "sess-1" {
		t.Errorf("SessionID = %q", req.SessionID)
	}
	if req.Mode != "sync" {
		t.Errorf("Mode = %q", req.Mode)
	}
	if len(req.Input) != 1 || req.Input[0].Parts[0].Content != "do something" {
		t.Errorf("Input = %+v", req.Input)
	}
}

func TestRunToEnvelope(t *testing.T) {
	run := &Run{
		RunID:     "run-1",
		AgentName: "echo",
		SessionID: "sess-1",
		Status:    RunStatusCompleted,
	}
	env := RunToEnvelope(run, "echo", "alice")
	if env.Protocol != protocol.ProtocolACP {
		t.Errorf("Protocol = %q", env.Protocol)
	}
	if env.Metadata["acp.run_id"] != "run-1" {
		t.Errorf("run_id = %q", env.Metadata["acp.run_id"])
	}
	if env.MessageType != envelope.MessageTypeResponse {
		t.Errorf("MessageType = %q", env.MessageType)
	}
}

func TestRunToEnvelope_Failed(t *testing.T) {
	run := &Run{RunID: "run-1", Status: RunStatusFailed}
	env := RunToEnvelope(run, "a", "b")
	if env.MessageType != envelope.MessageTypeError {
		t.Errorf("MessageType = %q", env.MessageType)
	}
}

func TestRunToEnvelope_Awaiting(t *testing.T) {
	run := &Run{RunID: "run-1", Status: RunStatusAwaiting}
	env := RunToEnvelope(run, "a", "b")
	if env.MessageType != envelope.MessageTypeEvent {
		t.Errorf("MessageType = %q", env.MessageType)
	}
}

func TestNewRun(t *testing.T) {
	req := CreateRunRequest{
		AgentName: "echo",
		SessionID: "sess-1",
		Input: []Message{
			{Role: "user", Parts: []MessagePart{{ContentType: "text/plain", Content: "test"}}},
		},
		Mode: "sync",
	}
	run := NewRun(req)
	if run.RunID == "" {
		t.Error("RunID should not be empty")
	}
	if run.AgentName != "echo" {
		t.Errorf("AgentName = %q", run.AgentName)
	}
	if run.Status != RunStatusCreated {
		t.Errorf("Status = %q", run.Status)
	}
}

func TestNewRun_AutoSession(t *testing.T) {
	req := CreateRunRequest{AgentName: "echo", Input: []Message{}}
	run := NewRun(req)
	if run.SessionID == "" {
		t.Error("SessionID should be auto-generated")
	}
}

func TestAgentManifestJSON(t *testing.T) {
	manifest := AgentManifest{
		Name:              "echo-agent",
		Description:       "An echo agent",
		InputContentTypes: []string{"text/plain"},
		Metadata: ManifestMetadata{
			Capabilities: []CapabilityDef{{Name: "echo"}},
			Tags:         []string{"test"},
		},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var decoded AgentManifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "echo-agent" {
		t.Errorf("Name = %q", decoded.Name)
	}
}
