package server

import "github.com/inspr-at/hausv-org/internal/store"

func issueRepositoryForTest(a *app, tenantSlug string) store.IssueRepository {
	repository, _ := store.BindIssueRepository(a.issueStore, testTenantRef(tenantSlug))
	return repository
}

func attachmentRepositoryForTest(a *app, tenantSlug string) store.AttachmentRepository {
	repository, _ := store.BindAttachmentRepository(a.attachmentStore, testTenantRef(tenantSlug))
	return repository
}

func documentRepositoryForTest(a *app, tenantSlug string) store.DocumentRepository {
	repository, _ := store.BindDocumentRepository(a.documentStore, testTenantRef(tenantSlug))
	return repository
}

func documentRepositoryForStorageTest(storage store.DocumentStorage, tenantSlug string) store.DocumentRepository {
	repository, _ := store.BindDocumentRepository(storage, testTenantRef(tenantSlug))
	return repository
}
