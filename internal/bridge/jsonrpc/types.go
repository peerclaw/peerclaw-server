package jsonrpc

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
)

const Version = "2.0"

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error represents a JSON-RPC 2.0 error object.
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

// Notification represents a JSON-RPC 2.0 notification (request with no ID).
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

var idCounter atomic.Int64

// NewRequest creates a new JSON-RPC request with an auto-incremented integer ID.
func NewRequest(method string, params any) (*Request, error) {
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		raw = b
	}
	return &Request{
		JSONRPC: Version,
		ID:      idCounter.Add(1),
		Method:  method,
		Params:  raw,
	}, nil
}

// NewRequestWithID creates a JSON-RPC request with a specific ID.
func NewRequestWithID(id any, method string, params any) (*Request, error) {
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		raw = b
	}
	return &Request{
		JSONRPC: Version,
		ID:      id,
		Method:  method,
		Params:  raw,
	}, nil
}

// NewResponse creates a successful JSON-RPC response.
func NewResponse(id any, result any) (*Response, error) {
	var raw json.RawMessage
	if result != nil {
		b, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal result: %w", err)
		}
		raw = b
	}
	return &Response{
		JSONRPC: Version,
		ID:      id,
		Result:  raw,
	}, nil
}

// NewErrorResponse creates an error JSON-RPC response.
func NewErrorResponse(id any, code int, message string) *Response {
	return &Response{
		JSONRPC: Version,
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
}

// NewNotification creates a JSON-RPC notification (no ID, no response expected).
func NewNotification(method string, params any) (*Notification, error) {
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		raw = b
	}
	return &Notification{
		JSONRPC: Version,
		Method:  method,
		Params:  raw,
	}, nil
}

// MessageKind indicates what type of JSON-RPC message was parsed.
type MessageKind int

const (
	KindRequest MessageKind = iota
	KindResponse
	KindNotification
)

// ParsedMessage holds the result of ParseMessage.
type ParsedMessage struct {
	Kind         MessageKind
	Request      *Request
	Response     *Response
	Notification *Notification
}

// ParseMessage decodes raw JSON and identifies whether it is a Request, Response, or Notification.
func ParseMessage(data []byte) (*ParsedMessage, error) {
	// Peek at the structure to determine type.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC message: %w", err)
	}

	if _, hasMethod := raw["method"]; hasMethod {
		// Has "method" — could be Request or Notification.
		if _, hasID := raw["id"]; hasID {
			var req Request
			if err := json.Unmarshal(data, &req); err != nil {
				return nil, fmt.Errorf("unmarshal request: %w", err)
			}
			return &ParsedMessage{Kind: KindRequest, Request: &req}, nil
		}
		var notif Notification
		if err := json.Unmarshal(data, &notif); err != nil {
			return nil, fmt.Errorf("unmarshal notification: %w", err)
		}
		return &ParsedMessage{Kind: KindNotification, Notification: &notif}, nil
	}

	// No "method" — must be a Response.
	if _, hasResult := raw["result"]; hasResult {
		var resp Response
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}
		return &ParsedMessage{Kind: KindResponse, Response: &resp}, nil
	}
	if _, hasError := raw["error"]; hasError {
		var resp Response
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("unmarshal error response: %w", err)
		}
		return &ParsedMessage{Kind: KindResponse, Response: &resp}, nil
	}

	return nil, fmt.Errorf("unrecognized JSON-RPC message: missing method, result, or error")
}
