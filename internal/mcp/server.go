package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
)

type CreateChildSpecArgs struct {
	ParentID string `json:"parent_id" jsonschema:"ID of the parent spec to create child under"`
	Title    string `json:"title" jsonschema:"Title of the new child spec"`
	Content  string `json:"content" jsonschema:"Content/description of the new child spec"`
}

type CreateChildSpecResult struct {
	ChildID  string `json:"child_id"`
	ParentID string `json:"parent_id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Message  string `json:"message"`
}

type Server struct {
	specService services.SpecService
	mcpServer   *mcp.Server
}

func NewServer(specService services.SpecService) *Server {
	return &Server{
		specService: specService,
	}
}

func (s *Server) CreateChildSpec(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[CreateChildSpecArgs]) (*mcp.CallToolResultFor[CreateChildSpecResult], error) {
	args := params.Arguments

	parentNode, err := s.specService.GetNode(args.ParentID)
	if err != nil {
		return &mcp.CallToolResultFor[CreateChildSpecResult]{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error: Parent spec with ID '%s' not found: %v", args.ParentID, err)},
			},
		}, nil
	}

	childSpec, err := s.specService.CreateSpec(args.Title, args.Content)
	if err != nil {
		return &mcp.CallToolResultFor[CreateChildSpecResult]{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error creating child spec: %v", err)},
			},
		}, nil
	}

	_, err = s.specService.AddChildToParent(childSpec.GetID(), args.ParentID, "child")
	if err != nil {
		return &mcp.CallToolResultFor[CreateChildSpecResult]{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error linking child to parent: %v", err)},
			},
		}, nil
	}

	result := CreateChildSpecResult{
		ChildID:  childSpec.GetID(),
		ParentID: args.ParentID,
		Title:    childSpec.GetTitle(),
		Content:  childSpec.GetContent(),
		Message:  fmt.Sprintf("Successfully created child spec '%s' under parent '%s'", childSpec.GetTitle(), parentNode.GetTitle()),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return &mcp.CallToolResultFor[CreateChildSpecResult]{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error marshaling result: %v", err)},
			},
		}, nil
	}

	return &mcp.CallToolResultFor[CreateChildSpecResult]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *Server) Start(transport string, address string) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "zamm-spec-server"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_child_spec",
		Description: "Create a new specification as a child of an existing specification",
	}, s.CreateChildSpec)

	s.mcpServer = server

	switch transport {
	case "stdio":
		log.Println("Starting MCP server with stdio transport")
		stdioTransport := mcp.NewStdioTransport()
		loggingTransport := mcp.NewLoggingTransport(stdioTransport, os.Stderr)
		if err := server.Run(context.Background(), loggingTransport); err != nil {
			return fmt.Errorf("server failed: %v", err)
		}
		return nil
	case "http":
		log.Printf("Starting MCP server with HTTP transport on %s", address)
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		return http.ListenAndServe(address, handler)
	default:
		return fmt.Errorf("unsupported transport type: %s", transport)
	}
}

func (s *Server) Stop() error {
	return nil
}
