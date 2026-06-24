package mcp

import (
	"encoding/json"
	"fmt"
)

const (
	ParseError       = -32700
	InvalidRequest   = -32600
	MethodNotFound   = -32601
	InvalidParams    = -32602
	InternalError    = -32603
	DuplicateIDError = -32610
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  any         `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("error %d: %s", e.Code, e.Message)
}

type RequestTracker struct {
	usedIDs map[any]bool
}

func NewRequestTracker() *RequestTracker {
	return &RequestTracker{usedIDs: make(map[any]bool)}
}

func (rt *RequestTracker) IsIDUsed(id any) bool {
	return rt.usedIDs[id]
}

func (rt *RequestTracker) MarkIDUsed(id any) {
	rt.usedIDs[id] = true
}

func (rt *RequestTracker) Reset() {
	rt.usedIDs = make(map[any]bool)
}

type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion,omitempty"`
	Capabilities    ClientCapabilities `json:"capabilities,omitempty"`
	ClientInfo      *ClientInfo        `json:"clientInfo,omitempty"`
}

type ClientCapabilities struct {
	Roots    *RootsCapability `json:"roots,omitempty"`
	Sampling *struct{}        `json:"sampling,omitempty"`
}

type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type ServerCapabilities struct {
	Tools *struct{} `json:"tools"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string          `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

func (is InputSchema) MarshalJSON() ([]byte, error) {
	type Alias InputSchema
	aux := struct {
		Alias
		Properties map[string]Property `json:"properties"`
	}{
		Alias:      Alias(is),
		Properties: is.Properties,
	}
	return json.Marshal(aux)
}

func NewRPCError(code int, msg string) *RPCError {
	return &RPCError{Code: code, Message: msg}
}

func NewInitializeResult(name, version string) *InitializeResult {
	return &InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &struct{}{},
		},
		ServerInfo: ServerInfo{
			Name:    name,
			Version: version,
		},
	}
}

func NewListToolsResult(tools []Tool) *ListToolsResult {
	return &ListToolsResult{Tools: tools}
}

type CallToolResult struct {
	Content           []TextContent `json:"content"`
	IsError           bool          `json:"isError,omitempty"`
	StructuredContent any           `json:"structuredContent,omitempty"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func NewCallToolResult(text string) *CallToolResult {
	return &CallToolResult{
		Content:           []TextContent{{Type: "text", Text: text}},
		IsError:           false,
		StructuredContent: nil,
	}
}

func NewCallToolResultWithStructured(text string, structured any) *CallToolResult {
	return &CallToolResult{
		Content:           []TextContent{{Type: "text", Text: text}},
		IsError:           false,
		StructuredContent: structured,
	}
}

func NewCallToolError(text string) *CallToolResult {
	return &CallToolResult{
		Content:           []TextContent{{Type: "text", Text: text}},
		IsError:           true,
		StructuredContent: nil,
	}
}

func ParseRequest(data []byte) (*JSONRPCRequest, error) {
	var req JSONRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if req.ID == nil {
		return nil, NewRPCError(InvalidRequest, "id MUST NOT be null")
	}

	return &req, nil
}

func MarshalResponse(resp JSONRPCResponse) []byte {
	data, _ := json.Marshal(resp)
	return data
}
