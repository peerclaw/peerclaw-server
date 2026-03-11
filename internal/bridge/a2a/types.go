package a2a

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
)

// TaskState represents the lifecycle state of an A2A task.
type TaskState string

const (
	TaskStateAccepted      TaskState = "accepted"
	TaskStateWorking       TaskState = "working"
	TaskStateCompleted     TaskState = "completed"
	TaskStateFailed        TaskState = "failed"
	TaskStateCanceled      TaskState = "canceled"
	TaskStateRejected      TaskState = "rejected"
	TaskStateInputRequired TaskState = "input_required"
	TaskStateAuthRequired  TaskState = "auth_required"
)

// Task represents an A2A task.
type Task struct {
	ID        string         `json:"id"`
	ContextID string         `json:"contextId"`
	Status    TaskStatus     `json:"status"`
	History   []Message      `json:"history,omitempty"`
	Artifacts []Artifact     `json:"artifacts,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"createdAt"`
	UpdatedAt string         `json:"updatedAt"`
}

// TaskStatus represents the current status of a task.
type TaskStatus struct {
	State     TaskState `json:"state"`
	Message   string    `json:"message,omitempty"`
	Timestamp string    `json:"timestamp"`
}

// Message represents an A2A message.
type Message struct {
	Role      string `json:"role"` // user, agent
	Parts     []Part `json:"parts"`
	MessageID string `json:"messageId,omitempty"`
	ContextID string `json:"contextId,omitempty"`
}

// Part represents a content part within a message.
type Part struct {
	Text           string         `json:"text,omitempty"`
	FileReference  *FileRef       `json:"fileReference,omitempty"`
	StructuredData map[string]any `json:"structuredData,omitempty"`
}

// FileRef represents a reference to a file.
type FileRef struct {
	URL       string `json:"url"`
	MediaType string `json:"mediaType,omitempty"`
	Size      int64  `json:"size,omitempty"`
}

// Artifact represents an output artifact from a task.
type Artifact struct {
	ID        string `json:"id"`
	MediaType string `json:"mediaType,omitempty"`
	Parts     []Part `json:"parts"`
}

// AgentCard represents an A2A-standard agent card.
type AgentCard struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Version      string   `json:"version,omitempty"`
	Endpoint     string   `json:"endpoint"`
	Capabilities A2ACaps  `json:"capabilities"`
	Skills       []Skill  `json:"skills,omitempty"`
}

// A2ACaps defines A2A protocol capabilities.
type A2ACaps struct {
	Streaming         bool `json:"streaming"`
	PushNotifications bool `json:"pushNotifications"`
	MultiTurn         bool `json:"multiTurn"`
}

// Skill describes a skill the agent can perform.
type Skill struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	InputModes  []string `json:"inputModes,omitempty"`
	OutputModes []string `json:"outputModes,omitempty"`
}

// PushNotificationConfig holds A2A push notification settings.
type PushNotificationConfig struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`
}

// SetPushNotificationParams is the JSON-RPC params for tasks/pushNotification/set.
type SetPushNotificationParams struct {
	ID                     string                 `json:"id"`
	PushNotificationConfig PushNotificationConfig `json:"pushNotificationConfig"`
}

// GetPushNotificationParams is the JSON-RPC params for tasks/pushNotification/get.
type GetPushNotificationParams struct {
	ID string `json:"id"`
}

// SendMessageParams is the JSON-RPC params for SendMessage.
type SendMessageParams struct {
	Message Message            `json:"message"`
	Config  *SendMessageConfig `json:"configuration,omitempty"`
}

// SendMessageConfig holds optional configuration for a SendMessage call.
type SendMessageConfig struct {
	AcceptedOutputModes []string `json:"acceptedOutputModes,omitempty"`
	Blocking            bool     `json:"blocking,omitempty"`
	HistoryLength       int      `json:"historyLength,omitempty"`
}

// GetTaskParams is the JSON-RPC params for GetTask.
type GetTaskParams struct {
	ID            string `json:"id"`
	HistoryLength int    `json:"historyLength,omitempty"`
}

// CancelTaskParams is the JSON-RPC params for CancelTask.
type CancelTaskParams struct {
	ID string `json:"id"`
}

// EnvelopeToMessage converts an Envelope payload to an A2A Message.
func EnvelopeToMessage(env *envelope.Envelope) Message {
	msg := Message{
		Role:      "user",
		MessageID: env.ID,
		ContextID: env.Metadata["a2a.context_id"],
		Parts: []Part{
			{Text: string(env.Payload)},
		},
	}
	return msg
}

// TaskToEnvelope converts an A2A Task into an Envelope.
func TaskToEnvelope(task *Task, source, destination string) *envelope.Envelope {
	payload, _ := json.Marshal(task)
	env := envelope.New(source, destination, protocol.ProtocolA2A, payload)
	env.Metadata["a2a.task_id"] = task.ID
	env.Metadata["a2a.context_id"] = task.ContextID
	env.Metadata["a2a.state"] = string(task.Status.State)

	switch task.Status.State {
	case TaskStateCompleted, TaskStateFailed, TaskStateCanceled, TaskStateRejected:
		env.MessageType = envelope.MessageTypeResponse
	case TaskStateInputRequired, TaskStateAuthRequired:
		env.MessageType = envelope.MessageTypeEvent
	default:
		env.MessageType = envelope.MessageTypeEvent
	}
	return env
}

// NewTask creates a new A2A Task from a message.
func NewTask(contextID string, msg Message) *Task {
	now := time.Now().UTC().Format(time.RFC3339)
	return &Task{
		ID:        uuid.New().String(),
		ContextID: contextID,
		Status: TaskStatus{
			State:     TaskStateAccepted,
			Timestamp: now,
		},
		History:   []Message{msg},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
