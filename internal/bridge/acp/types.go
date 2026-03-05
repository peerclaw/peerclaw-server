package acp

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
)

// RunStatus represents the lifecycle status of an ACP run.
type RunStatus string

const (
	RunStatusCreated    RunStatus = "created"
	RunStatusInProgress RunStatus = "in-progress"
	RunStatusAwaiting   RunStatus = "awaiting"
	RunStatusCompleted  RunStatus = "completed"
	RunStatusFailed     RunStatus = "failed"
	RunStatusCancelling RunStatus = "cancelling"
	RunStatusCancelled  RunStatus = "cancelled"
)

// Run represents an ACP run.
type Run struct {
	AgentName string    `json:"agent_name"`
	RunID     string    `json:"run_id"`
	SessionID string    `json:"session_id,omitempty"`
	Status    RunStatus `json:"status"`
	Input     []Message `json:"input,omitempty"`
	Output    []Message `json:"output,omitempty"`
	Error     *RunError `json:"error,omitempty"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

// RunError holds error details for a failed run.
type RunError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

// Message represents an ACP message.
type Message struct {
	Role      string        `json:"role"` // user, agent, agent/{name}
	Parts     []MessagePart `json:"parts"`
	CreatedAt string        `json:"created_at,omitempty"`
}

// MessagePart represents a part of an ACP message.
type MessagePart struct {
	ContentType     string         `json:"content_type"`
	Content         string         `json:"content,omitempty"`
	ContentURL      string         `json:"content_url,omitempty"`
	ContentEncoding string         `json:"content_encoding,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// AgentManifest describes an ACP-compatible agent.
type AgentManifest struct {
	Name               string           `json:"name"`
	Description        string           `json:"description,omitempty"`
	InputContentTypes  []string         `json:"input_content_types,omitempty"`
	OutputContentTypes []string         `json:"output_content_types,omitempty"`
	Metadata           ManifestMetadata `json:"metadata,omitempty"`
}

// ManifestMetadata holds structured metadata for an agent manifest.
type ManifestMetadata struct {
	Capabilities []CapabilityDef `json:"capabilities,omitempty"`
	Domains      []string        `json:"domains,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
}

// CapabilityDef describes a capability.
type CapabilityDef struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CreateRunRequest is the request body for POST /acp/runs.
type CreateRunRequest struct {
	AgentName string    `json:"agent_name"`
	SessionID string    `json:"session_id,omitempty"`
	Input     []Message `json:"input"`
	Mode      string    `json:"mode"` // sync, async, stream
}

// Session tracks a group of runs.
type Session struct {
	ID     string
	RunIDs []string
}

// EnvelopeToCreateRun converts an Envelope to a CreateRunRequest.
func EnvelopeToCreateRun(env *envelope.Envelope) CreateRunRequest {
	agentName := env.Metadata["acp.agent_name"]
	if agentName == "" {
		agentName = env.Destination
	}
	sessionID := env.Metadata["acp.session_id"]
	mode := env.Metadata["acp.mode"]
	if mode == "" {
		mode = "sync"
	}

	return CreateRunRequest{
		AgentName: agentName,
		SessionID: sessionID,
		Input: []Message{
			{
				Role: "user",
				Parts: []MessagePart{
					{
						ContentType: "text/plain",
						Content:     string(env.Payload),
					},
				},
			},
		},
		Mode: mode,
	}
}

// RunToEnvelope converts an ACP Run to an Envelope.
func RunToEnvelope(run *Run, source, dest string) *envelope.Envelope {
	payload, _ := json.Marshal(run)
	env := envelope.New(source, dest, protocol.ProtocolACP, payload)
	env.Metadata["acp.run_id"] = run.RunID
	env.Metadata["acp.session_id"] = run.SessionID
	env.Metadata["acp.agent_name"] = run.AgentName

	switch run.Status {
	case RunStatusCompleted:
		env.MessageType = envelope.MessageTypeResponse
	case RunStatusFailed:
		env.MessageType = envelope.MessageTypeError
	case RunStatusAwaiting:
		env.MessageType = envelope.MessageTypeEvent
	default:
		env.MessageType = envelope.MessageTypeEvent
	}
	return env
}

// NewRun creates a new Run from a CreateRunRequest.
func NewRun(req CreateRunRequest) *Run {
	now := time.Now().UTC().Format(time.RFC3339)
	runID := uuid.New().String()
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}
	return &Run{
		AgentName: req.AgentName,
		RunID:     runID,
		SessionID: sessionID,
		Status:    RunStatusCreated,
		Input:     req.Input,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
