package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

// handleResolveRepo serves the scoped and by-ID lookups ResolveHandle makes.
type handleResolveRepo struct {
	interfaces.TemporaryDocumentRepository
	documents []*types.TemporaryDocument
}

func (r *handleResolveRepo) GetScoped(
	_ context.Context, tenantID uint64, sessionID, documentID string,
) (*types.TemporaryDocument, error) {
	for _, document := range r.documents {
		if document.TenantID == tenantID && document.SessionID == sessionID && document.ID == documentID {
			return document, nil
		}
	}
	return nil, nil
}

func (r *handleResolveRepo) GetByID(
	_ context.Context, tenantID uint64, documentID string,
) (*types.TemporaryDocument, error) {
	for _, document := range r.documents {
		if document.TenantID == tenantID && document.ID == documentID {
			return document, nil
		}
	}
	return nil, nil
}

func (r *handleResolveRepo) ListScoped(
	_ context.Context, tenantID uint64, sessionID string,
) ([]*types.TemporaryDocument, error) {
	scoped := make([]*types.TemporaryDocument, 0, len(r.documents))
	for _, document := range r.documents {
		if document.TenantID == tenantID && document.SessionID == sessionID {
			scoped = append(scoped, document)
		}
	}
	return scoped, nil
}

func attachmentDocument(id, sessionID string) *types.TemporaryDocument {
	return &types.TemporaryDocument{
		ID:        id,
		TenantID:  7,
		SessionID: sessionID,
		FileName:  "sales.csv",
		FileType:  ".csv",
		Status:    types.TemporaryDocumentStatusReady,
	}
}

func TestResolveHandleFindsSessionDocument(t *testing.T) {
	document := attachmentDocument("doc-1", "session-1")
	svc := &temporaryDocumentService{
		repo: &handleResolveRepo{documents: []*types.TemporaryDocument{document}},
	}

	got, err := svc.ResolveHandle(context.Background(), 7, "session-1", types.AttachmentPath("doc-1"))

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "doc-1", got.ID)
}

func TestResolveHandleResolvesParentAttachmentOnFork(t *testing.T) {
	document := attachmentDocument("doc-1", "parent")
	svc := &temporaryDocumentService{
		// ListScoped panics here: the handle must resolve through the fork's
		// message references, exactly like Get does.
		repo: &forkAccessDocRepo{docs: []*types.TemporaryDocument{document}},
		sessionAttachments: &forkAccessAttachments{bySession: map[string]types.MessageAttachments{
			"fork": {{ID: "doc-1", FileName: "sales.csv"}},
		}},
	}

	got, err := svc.ResolveHandle(context.Background(), 7, "fork", "attachment://"+types.AttachmentHandle("doc-1"))

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "doc-1", got.ID)
}

func TestResolveHandleFallsBackToSessionListing(t *testing.T) {
	document := attachmentDocument("doc-2", "session-1")
	svc := &temporaryDocumentService{
		repo: &handleResolveRepo{documents: []*types.TemporaryDocument{document}},
	}

	got, err := svc.ResolveHandle(context.Background(), 7, "session-1", types.AttachmentHandle("doc-2"))

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "doc-2", got.ID)
}

func TestResolveHandleIgnoresAnotherConversationsAttachment(t *testing.T) {
	document := attachmentDocument("doc-3", "other-session")
	svc := &temporaryDocumentService{
		repo: &handleResolveRepo{documents: []*types.TemporaryDocument{document}},
	}

	got, err := svc.ResolveHandle(context.Background(), 7, "session-1", types.AttachmentPath("doc-3"))

	require.NoError(t, err)
	require.Nil(t, got)
}

func TestResolveHandleAcceptsRawDocumentID(t *testing.T) {
	document := attachmentDocument("doc-4", "session-1")
	svc := &temporaryDocumentService{
		repo: &handleResolveRepo{documents: []*types.TemporaryDocument{document}},
	}

	got, err := svc.ResolveHandle(context.Background(), 7, "session-1", "doc-4")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "doc-4", got.ID)
}
