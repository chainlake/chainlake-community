package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// conversationAttachments is the TemporaryDocumentService surface data_analysis
// needs for a conversation upload: model-facing handle resolution plus the raw
// bytes of the stored file.
type conversationAttachments struct {
	interfaces.TemporaryDocumentService
	documents map[string]*types.TemporaryDocument
	files     map[string]string
}

func (c *conversationAttachments) ResolveHandle(
	_ context.Context, _ uint64, _ string, handle string,
) (*types.TemporaryDocument, error) {
	ref := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(handle), types.AttachmentPathScheme), "attachment:")
	for _, document := range c.documents {
		if document.ID == ref || types.AttachmentHandle(document.ID) == ref {
			return document, nil
		}
	}
	return nil, nil
}

func (c *conversationAttachments) OpenFile(
	_ context.Context, _ uint64, _ string, documentID string,
) (io.ReadCloser, string, error) {
	content, ok := c.files[documentID]
	if !ok {
		return nil, "", fmt.Errorf("no such attachment %q", documentID)
	}
	return io.NopCloser(strings.NewReader(content)), documentID, nil
}

func tenantContext(tenantID uint64) context.Context {
	return context.WithValue(context.Background(), types.TenantIDContextKey, tenantID)
}

func conversationCSVAttachment() (*types.TemporaryDocument, *conversationAttachments) {
	document := &types.TemporaryDocument{
		ID:        "att-doc-1",
		TenantID:  7,
		SessionID: "session-1",
		FileName:  "sales.csv",
		FileType:  ".csv",
		Status:    types.TemporaryDocumentStatusReady,
	}
	return document, &conversationAttachments{
		documents: map[string]*types.TemporaryDocument{document.ID: document},
		files:     map[string]string{document.ID: "id,amount\n1,10\n2,20\n"},
	}
}

func TestDataAnalysisRunsSQLOnConversationAttachment(t *testing.T) {
	ctx := tenantContext(7)
	document, attachments := conversationCSVAttachment()
	tool := NewDataAnalysisTool(nil, nil, nil, nil, newPlainDuckDB(t), "session-1").
		WithTemporaryDocuments(attachments)
	registry := NewToolRegistry()
	registry.RegisterTool(tool)
	t.Cleanup(func() { tool.Cleanup(context.Background()) })

	args, err := json.Marshal(DataAnalysisInput{
		AttachmentID: types.AttachmentPath(document.ID),
		SQL:          `SELECT SUM(CAST(amount AS INTEGER)) AS total FROM dataset`,
	})
	require.NoError(t, err)

	result, err := registry.ExecuteTool(ctx, ToolDataAnalysis, args)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success, result.Error)
	assert.Contains(t, result.Output, `"total":"30"`)
}

func TestDataAnalysisAttachmentAcceptsRawDocumentID(t *testing.T) {
	ctx := tenantContext(7)
	document, attachments := conversationCSVAttachment()
	tool := NewDataAnalysisTool(nil, nil, nil, nil, newPlainDuckDB(t), "session-1").
		WithTemporaryDocuments(attachments)
	registry := NewToolRegistry()
	registry.RegisterTool(tool)
	t.Cleanup(func() { tool.Cleanup(context.Background()) })

	args, err := json.Marshal(DataAnalysisInput{
		AttachmentID: document.ID,
		SQL:          `SELECT COUNT(*) AS total FROM dataset`,
	})
	require.NoError(t, err)

	result, err := registry.ExecuteTool(ctx, ToolDataAnalysis, args)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success, result.Error)
	assert.Contains(t, result.Output, `"total":"2"`)
}

func TestDataAnalysisRejectsUnknownAttachmentHandle(t *testing.T) {
	ctx := tenantContext(7)
	_, attachments := conversationCSVAttachment()
	tool := NewDataAnalysisTool(nil, nil, nil, nil, newPlainDuckDB(t), "session-1").
		WithTemporaryDocuments(attachments)

	args, err := json.Marshal(DataAnalysisInput{
		AttachmentID: "attachment://att_deadbeef",
		SQL:          `SELECT * FROM dataset`,
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "was not found in this conversation")
}

func TestDataAnalysisRejectsNonTabularAttachment(t *testing.T) {
	ctx := tenantContext(7)
	document := &types.TemporaryDocument{
		ID: "att-pdf", TenantID: 7, SessionID: "session-1",
		FileName: "report.pdf", FileType: ".pdf", Status: types.TemporaryDocumentStatusReady,
	}
	attachments := &conversationAttachments{
		documents: map[string]*types.TemporaryDocument{document.ID: document},
		files:     map[string]string{document.ID: "%PDF-1.7"},
	}
	tool := NewDataAnalysisTool(nil, nil, nil, nil, newPlainDuckDB(t), "session-1").
		WithTemporaryDocuments(attachments)

	args, err := json.Marshal(DataAnalysisInput{
		AttachmentID: types.AttachmentPath(document.ID),
		SQL:          `SELECT * FROM dataset`,
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "data analysis reads csv, xlsx or xls only")
}

func TestDataAnalysisAttachmentRequiresExactlyOneSource(t *testing.T) {
	ctx := tenantContext(7)
	_, attachments := conversationCSVAttachment()
	tool := NewDataAnalysisTool(nil, nil, nil, nil, newPlainDuckDB(t), "session-1").
		WithTemporaryDocuments(attachments)

	for name, input := range map[string]DataAnalysisInput{
		"neither": {SQL: `SELECT * FROM dataset`},
		"both":    {KnowledgeID: "doc-1", AttachmentID: "attachment://att_deadbeef", SQL: `SELECT * FROM dataset`},
	} {
		t.Run(name, func(t *testing.T) {
			args, err := json.Marshal(input)
			require.NoError(t, err)
			result, err := tool.Execute(ctx, args)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.False(t, result.Success)
		})
	}
}

func TestIsTabularDataFile(t *testing.T) {
	assert.True(t, IsTabularDataFile(".csv", "sales.csv"))
	assert.True(t, IsTabularDataFile("xlsx", "sales.xlsx"))
	assert.True(t, IsTabularDataFile("", "sales.XLS"))
	assert.False(t, IsTabularDataFile(".pdf", "report.pdf"))
	assert.False(t, IsTabularDataFile("", ""))
}
