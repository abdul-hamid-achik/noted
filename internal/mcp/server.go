package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/veclite"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is set by ldflags at build time
var Version = "dev"

// Server handles MCP protocol using the official Go SDK.
type Server struct {
	queries *db.Queries
	server  *mcp.Server
	syncer  Syncer
}

// Syncer interface for optional semantic search integration
type Syncer interface {
	Search(query string, limit int) ([]veclite.SemanticResult, error)
	SyncNote(id int64, title, content string) error
	Delete(noteID int64) error
	Close() error
}

// NewServer creates a new MCP server instance
func NewServer(queries *db.Queries, syncer Syncer) *Server {
	return &Server{
		queries: queries,
		syncer:  syncer,
	}
}

// Run starts the MCP server with stdio transport
func (s *Server) Run(ctx context.Context) error {
	// Create MCP server with implementation info
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    "noted",
		Version: Version,
	}, nil)

	// Register tools
	s.registerTools()

	// Run with stdio transport
	transport := &mcp.StdioTransport{}
	if err := s.server.Run(ctx, transport); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		return err
	}

	return nil
}

// HasSemanticSearch returns true if veclite integration is available
func (s *Server) HasSemanticSearch() bool {
	return s.syncer != nil
}
