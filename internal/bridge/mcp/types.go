package mcp

import (
	"encoding/json"

	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
)

// InitializeParams is the params for the initialize request.
type InitializeParams struct {
	ProtocolVersion string        `json:"protocolVersion"`
	Capabilities    ClientCaps    `json:"capabilities"`
	ClientInfo      ImplementInfo `json:"clientInfo"`
}

// InitializeResult is the result of the initialize request.
type InitializeResult struct {
	ProtocolVersion string        `json:"protocolVersion"`
	Capabilities    ServerCaps    `json:"capabilities"`
	ServerInfo      ImplementInfo `json:"serverInfo"`
	Instructions    string        `json:"instructions,omitempty"`
}

// ImplementInfo identifies an MCP client or server.
type ImplementInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCaps describes what the MCP client supports.
type ClientCaps struct {
	Roots    *RootsCap `json:"roots,omitempty"`
	Sampling any       `json:"sampling,omitempty"`
}

// ServerCaps describes what the MCP server supports.
type ServerCaps struct {
	Tools     *ToolsCap     `json:"tools,omitempty"`
	Resources *ResourcesCap `json:"resources,omitempty"`
	Prompts   *PromptsCap   `json:"prompts,omitempty"`
	Logging   any           `json:"logging,omitempty"`
}

// ToolsCap describes tools capabilities.
type ToolsCap struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCap describes resources capabilities.
type ResourcesCap struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCap describes prompts capabilities.
type PromptsCap struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// RootsCap describes roots capabilities.
type RootsCap struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolDef defines an MCP tool.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// ToolCallParams is the params for tools/call.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolCallResult is the result of tools/call.
type ToolCallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents content in MCP messages.
type Content struct {
	Type     string `json:"type"` // text, image, resource
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// Resource represents an MCP resource.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceContent holds the content of a resource.
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// ResourceReadParams is the params for resources/read.
type ResourceReadParams struct {
	URI string `json:"uri"`
}

// ResourceReadResult is the result of resources/read.
type ResourceReadResult struct {
	Contents []ResourceContent `json:"contents"`
}

// Prompt represents an MCP prompt template.
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument describes a prompt argument.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptMessage is a message in a prompt response.
type PromptMessage struct {
	Role    string  `json:"role"`
	Content Content `json:"content"`
}

// PromptGetParams is the params for prompts/get.
type PromptGetParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// PromptGetResult is the result of prompts/get.
type PromptGetResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// ToolsListResult is the result of tools/list.
type ToolsListResult struct {
	Tools []ToolDef `json:"tools"`
}

// ResourcesListResult is the result of resources/list.
type ResourcesListResult struct {
	Resources []Resource `json:"resources"`
}

// PromptsListResult is the result of prompts/list.
type PromptsListResult struct {
	Prompts []Prompt `json:"prompts"`
}

// EnvelopeToToolCall converts an Envelope to MCP ToolCallParams.
func EnvelopeToToolCall(env *envelope.Envelope) (*ToolCallParams, error) {
	toolName := env.Metadata["mcp.tool_name"]
	if toolName == "" {
		// Try parsing from payload.
		var tc ToolCallParams
		if err := json.Unmarshal(env.Payload, &tc); err == nil && tc.Name != "" {
			return &tc, nil
		}
		return nil, nil
	}
	return &ToolCallParams{
		Name:      toolName,
		Arguments: json.RawMessage(env.Payload),
	}, nil
}

// ToolResultToEnvelope converts a ToolCallResult to an Envelope.
func ToolResultToEnvelope(result *ToolCallResult, source, dest string) *envelope.Envelope {
	payload, _ := json.Marshal(result)
	env := envelope.New(source, dest, protocol.ProtocolMCP, payload)
	env.MessageType = envelope.MessageTypeResponse
	if result.IsError {
		env.MessageType = envelope.MessageTypeError
	}
	return env
}
