package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/agent/tools"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sessionAttachmentDocuments exposes the conversation's stored uploads to the
// tool-registration gate.
type sessionAttachmentDocuments struct {
	interfaces.TemporaryDocumentService
	documents []*types.TemporaryDocument
}

func (s *sessionAttachmentDocuments) List(
	context.Context, uint64, string,
) ([]*types.TemporaryDocument, error) {
	return s.documents, nil
}

func registerDataAnalysisForSession(
	t *testing.T, sessionID string, documents ...*types.TemporaryDocument,
) []string {
	t.Helper()
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(7))
	svc := &agentService{temporaryDocuments: &sessionAttachmentDocuments{documents: documents}}
	registry := tools.NewToolRegistry()
	config := &types.AgentConfig{AllowedTools: []string{tools.ToolDataAnalysis}}

	require.NoError(t, svc.registerTools(ctx, registry, config, nil, nil, sessionID))

	return registry.ListTools()
}

func TestRegisterToolsKeepsDataAnalysisForUploadedCSV(t *testing.T) {
	registered := registerDataAnalysisForSession(t, "session-1", &types.TemporaryDocument{
		ID: "att-1", TenantID: 7, SessionID: "session-1", FileName: "sales.csv", FileType: ".csv",
	})

	assert.Contains(t, registered, tools.ToolDataAnalysis)
}

func TestRegisterToolsDropsDataAnalysisWithoutKnowledgeScopeOrTabularAttachment(t *testing.T) {
	registered := registerDataAnalysisForSession(t, "session-1", &types.TemporaryDocument{
		ID: "att-2", TenantID: 7, SessionID: "session-1", FileName: "report.pdf", FileType: ".pdf",
	})

	assert.NotContains(t, registered, tools.ToolDataAnalysis)
}
