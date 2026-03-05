package a2a

import (
	"encoding/json"
	"testing"

	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
)

func TestTaskStateValues(t *testing.T) {
	states := []TaskState{
		TaskStateAccepted, TaskStateWorking, TaskStateCompleted,
		TaskStateFailed, TaskStateCanceled, TaskStateRejected,
		TaskStateInputRequired, TaskStateAuthRequired,
	}
	for _, s := range states {
		if s == "" {
			t.Error("empty task state")
		}
	}
}

func TestTaskJSON(t *testing.T) {
	task := &Task{
		ID:        "task-1",
		ContextID: "ctx-1",
		Status: TaskStatus{
			State:     TaskStateWorking,
			Timestamp: "2025-01-01T00:00:00Z",
		},
		History: []Message{
			{
				Role:  "user",
				Parts: []Part{{Text: "hello"}},
			},
		},
		Artifacts: []Artifact{
			{
				ID:    "art-1",
				Parts: []Part{{Text: "result"}},
			},
		},
		CreatedAt: "2025-01-01T00:00:00Z",
		UpdatedAt: "2025-01-01T00:00:00Z",
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Task
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ID != "task-1" {
		t.Errorf("ID = %q", decoded.ID)
	}
	if decoded.Status.State != TaskStateWorking {
		t.Errorf("State = %q", decoded.Status.State)
	}
	if len(decoded.History) != 1 {
		t.Errorf("History len = %d", len(decoded.History))
	}
}

func TestEnvelopeToMessage(t *testing.T) {
	env := envelope.New("alice", "bob", protocol.ProtocolA2A, []byte("hello world"))
	env.Metadata["a2a.context_id"] = "ctx-123"

	msg := EnvelopeToMessage(env)
	if msg.Role != "user" {
		t.Errorf("Role = %q", msg.Role)
	}
	if msg.ContextID != "ctx-123" {
		t.Errorf("ContextID = %q", msg.ContextID)
	}
	if len(msg.Parts) != 1 || msg.Parts[0].Text != "hello world" {
		t.Errorf("Parts = %+v", msg.Parts)
	}
}

func TestTaskToEnvelope(t *testing.T) {
	task := &Task{
		ID:        "task-1",
		ContextID: "ctx-1",
		Status: TaskStatus{
			State:     TaskStateCompleted,
			Timestamp: "2025-01-01T00:00:00Z",
		},
	}

	env := TaskToEnvelope(task, "bob", "alice")
	if env.Protocol != protocol.ProtocolA2A {
		t.Errorf("Protocol = %q", env.Protocol)
	}
	if env.Metadata["a2a.task_id"] != "task-1" {
		t.Errorf("task_id = %q", env.Metadata["a2a.task_id"])
	}
	if env.MessageType != envelope.MessageTypeResponse {
		t.Errorf("MessageType = %q", env.MessageType)
	}
}

func TestNewTask(t *testing.T) {
	msg := Message{
		Role:  "user",
		Parts: []Part{{Text: "do something"}},
	}
	task := NewTask("ctx-1", msg)
	if task.ID == "" {
		t.Error("ID should not be empty")
	}
	if task.ContextID != "ctx-1" {
		t.Errorf("ContextID = %q", task.ContextID)
	}
	if task.Status.State != TaskStateAccepted {
		t.Errorf("State = %q", task.Status.State)
	}
	if len(task.History) != 1 {
		t.Errorf("History len = %d", len(task.History))
	}
}

func TestAgentCardJSON(t *testing.T) {
	card := AgentCard{
		Name:     "test-agent",
		Endpoint: "https://example.com/a2a",
		Capabilities: A2ACaps{
			Streaming: true,
			MultiTurn: true,
		},
		Skills: []Skill{
			{Name: "summarize", InputModes: []string{"text"}, OutputModes: []string{"text"}},
		},
	}

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}
	var decoded AgentCard
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "test-agent" {
		t.Errorf("Name = %q", decoded.Name)
	}
	if !decoded.Capabilities.Streaming {
		t.Error("expected streaming = true")
	}
	if len(decoded.Skills) != 1 {
		t.Errorf("Skills len = %d", len(decoded.Skills))
	}
}
