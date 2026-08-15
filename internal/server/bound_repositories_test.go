package server

import "github.com/inspr-at/hausv-org/internal/store"

func issueRepositoryForTest(a *app, tenantSlug string) store.IssueRepository {
	repository, _ := store.BindIssueRepository(a.issueStore, tenantSlug)
	return repository
}

func attachmentRepositoryForTest(a *app, tenantSlug string) store.AttachmentRepository {
	repository, _ := store.BindAttachmentRepository(a.attachmentStore, tenantSlug)
	return repository
}

func documentRepositoryForTest(a *app, tenantSlug string) store.DocumentRepository {
	repository, _ := store.BindDocumentRepository(a.documentStore, tenantSlug)
	return repository
}

func documentRepositoryForStorageTest(storage store.DocumentStorage, tenantSlug string) store.DocumentRepository {
	repository, _ := store.BindDocumentRepository(storage, tenantSlug)
	return repository
}
