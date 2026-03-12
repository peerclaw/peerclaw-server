package acp

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
	coreacp "github.com/peerclaw/peerclaw-core/protocol/acp"
)

// Type aliases — re-export shared ACP types from core (H-15).
type (
	RunStatus       = coreacp.RunStatus
	Run             = coreacp.Run
	RunError        = coreacp.RunError
	Message         = coreacp.Message
	MessagePart     = coreacp.MessagePart
	AgentManifest   = coreacp.AgentManifest
	ManifestMetadata = coreacp.ManifestMetadata
	CapabilityDef   = coreacp.CapabilityDef
	CreateRunRequest = coreacp.CreateRunRequest
)

// Re-export status constants for backward compatibility.
const (
	RunStatusCreated    = coreacp.RunStatusCreated
	RunStatusInProgress = coreacp.RunStatusInProgress
	RunStatusAwaiting   = coreacp.RunStatusAwaiting
	RunStatusCompleted  = coreacp.RunStatusCompleted
	RunStatusFailed     = coreacp.RunStatusFailed
	RunStatusCancelling = coreacp.RunStatusCancelling
	RunStatusCancelled  = coreacp.RunStatusCancelled
)

// Session tracks a group of runs (server-specific).
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
