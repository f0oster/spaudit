package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"spaudit/domain/audit"
	"spaudit/domain/contracts"
	"spaudit/domain/jobs"
	"spaudit/domain/sharepoint"
)

// MockSiteRepository implements SiteRepository for testing
type MockSiteRepository struct {
	mock.Mock
}

func (m *MockSiteRepository) GetByID(ctx context.Context, siteID int64) (*sharepoint.Site, error) {
	args := m.Called(ctx, siteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sharepoint.Site), args.Error(1)
}

func (m *MockSiteRepository) Save(ctx context.Context, site *sharepoint.Site) error {
	args := m.Called(ctx, site)
	return args.Error(0)
}

func (m *MockSiteRepository) ListAll(ctx context.Context) ([]*sharepoint.Site, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Site), args.Error(1)
}

func (m *MockSiteRepository) GetWithMetadata(ctx context.Context, siteID int64) (*contracts.SiteWithMetadata, error) {
	args := m.Called(ctx, siteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*contracts.SiteWithMetadata), args.Error(1)
}

func (m *MockSiteRepository) GetAllWithMetadata(ctx context.Context) ([]*contracts.SiteWithMetadata, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*contracts.SiteWithMetadata), args.Error(1)
}

// MockListRepository implements ListRepository for testing
type MockListRepository struct {
	mock.Mock
}

func (m *MockListRepository) Save(ctx context.Context, list *sharepoint.List) error {
	args := m.Called(ctx, list)
	return args.Error(0)
}

func (m *MockListRepository) GetByID(ctx context.Context, siteID int64, listID string) (*sharepoint.List, error) {
	args := m.Called(ctx, siteID, listID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sharepoint.List), args.Error(1)
}

func (m *MockListRepository) GetByWebID(ctx context.Context, siteID int64, webID string) ([]*sharepoint.List, error) {
	args := m.Called(ctx, siteID, webID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.List), args.Error(1)
}

func (m *MockListRepository) GetAllForSite(ctx context.Context, siteID int64) ([]*sharepoint.List, error) {
	args := m.Called(ctx, siteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.List), args.Error(1)
}

// MockJobRepository implements JobRepository for testing
type MockJobRepository struct {
	mock.Mock
}

func (m *MockJobRepository) GetLastAuditDate(ctx context.Context, siteID int64) (*time.Time, error) {
	args := m.Called(ctx, siteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

func (m *MockJobRepository) ListActiveJobs(ctx context.Context) ([]*jobs.Job, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*jobs.Job), args.Error(1)
}

func (m *MockJobRepository) CreateJob(ctx context.Context, job *jobs.Job) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockJobRepository) UpdateJobStatus(ctx context.Context, jobID string, status jobs.JobStatus, progress *jobs.JobProgress) error {
	args := m.Called(ctx, jobID, status, progress)
	return args.Error(0)
}

func (m *MockJobRepository) CompleteJob(ctx context.Context, jobID string, result string) error {
	args := m.Called(ctx, jobID, result)
	return args.Error(0)
}

func (m *MockJobRepository) FailJob(ctx context.Context, jobID string, errorMsg string) error {
	args := m.Called(ctx, jobID, errorMsg)
	return args.Error(0)
}

func (m *MockJobRepository) DeleteOldJobs(ctx context.Context, olderThan time.Time) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func (m *MockJobRepository) CancelJob(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockJobRepository) GetJob(ctx context.Context, jobID string) (*jobs.Job, error) {
	args := m.Called(ctx, jobID)
	return args.Get(0).(*jobs.Job), args.Error(1)
}

func (m *MockJobRepository) ListJobs(ctx context.Context) ([]*jobs.Job, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*jobs.Job), args.Error(1)
}

func (m *MockJobRepository) ListJobsByType(ctx context.Context, jobType jobs.JobType) ([]*jobs.Job, error) {
	args := m.Called(ctx, jobType)
	return args.Get(0).([]*jobs.Job), args.Error(1)
}

func (m *MockJobRepository) ListJobsByStatus(ctx context.Context, status jobs.JobStatus) ([]*jobs.Job, error) {
	args := m.Called(ctx, status)
	return args.Get(0).([]*jobs.Job), args.Error(1)
}

func (m *MockJobRepository) UpdateJob(ctx context.Context, job *jobs.Job) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

// MockAssignmentRepository implements AssignmentRepository for testing
type MockAssignmentRepository struct {
	mock.Mock
}

func (m *MockAssignmentRepository) GetAssignmentsForObject(ctx context.Context, siteID int64, objectType, objectKey string) ([]*sharepoint.Assignment, error) {
	args := m.Called(ctx, siteID, objectType, objectKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Assignment), args.Error(1)
}

func (m *MockAssignmentRepository) GetResolvedAssignmentsForObject(ctx context.Context, siteID int64, objectType, objectKey string) ([]*sharepoint.ResolvedAssignment, error) {
	args := m.Called(ctx, siteID, objectType, objectKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.ResolvedAssignment), args.Error(1)
}

// MockItemRepository implements ItemRepository for testing
type MockItemRepository struct {
	mock.Mock
}

func (m *MockItemRepository) GetItemsForList(ctx context.Context, siteID int64, listID string, offset, limit int64) ([]*sharepoint.Item, error) {
	args := m.Called(ctx, siteID, listID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Item), args.Error(1)
}

func (m *MockItemRepository) GetItemsWithUniqueForList(ctx context.Context, siteID int64, listID string, offset, limit int64) ([]*sharepoint.Item, error) {
	args := m.Called(ctx, siteID, listID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Item), args.Error(1)
}

// MockSharingRepository implements SharingRepository for testing
type MockSharingRepository struct {
	mock.Mock
}

func (m *MockSharingRepository) GetSharingLinksForList(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLink, error) {
	args := m.Called(ctx, siteID, listID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.SharingLink), args.Error(1)
}

func (m *MockSharingRepository) GetSharingLinksWithItemDataForList(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLinkWithItemData, error) {
	args := m.Called(ctx, siteID, listID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.SharingLinkWithItemData), args.Error(1)
}

func (m *MockSharingRepository) GetSharingLinkMembers(ctx context.Context, siteID int64, linkID string) ([]*sharepoint.Principal, error) {
	args := m.Called(ctx, siteID, linkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Principal), args.Error(1)
}

// MockAuditService implements AuditService for testing
type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) IsSiteBeingAudited(siteURL string) bool {
	args := m.Called(siteURL)
	return args.Bool(0)
}

func (m *MockAuditService) BuildAuditParametersFromFormData(formData map[string][]string) *audit.AuditParameters {
	args := m.Called(formData)
	return args.Get(0).(*audit.AuditParameters)
}

func (m *MockAuditService) QueueAudit(ctx context.Context, siteURL string, parameters *audit.AuditParameters) (*audit.AuditRequest, error) {
	args := m.Called(ctx, siteURL, parameters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*audit.AuditRequest), args.Error(1)
}

func (m *MockAuditService) GetAuditStatus(siteURL string) (*audit.ActiveAudit, bool) {
	args := m.Called(siteURL)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*audit.ActiveAudit), args.Bool(1)
}

func (m *MockAuditService) GetActiveAudits() []*audit.ActiveAudit {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]*audit.ActiveAudit)
}

func (m *MockAuditService) GetAuditHistory(limit int) []*audit.AuditResult {
	args := m.Called(limit)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]*audit.AuditResult)
}

func (m *MockAuditService) CancelAudit(siteURL string) error {
	args := m.Called(siteURL)
	return args.Error(0)
}

// MockJobServiceForApplication implements JobService interface for application layer testing
type MockJobServiceForApplication struct {
	mock.Mock
}

func (m *MockJobServiceForApplication) CreateJob(jobType jobs.JobType, siteURL, description string) (*jobs.Job, error) {
	args := m.Called(jobType, siteURL, description)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jobs.Job), args.Error(1)
}

func (m *MockJobServiceForApplication) GetJob(jobID string) (*jobs.Job, bool) {
	args := m.Called(jobID)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*jobs.Job), args.Bool(1)
}

func (m *MockJobServiceForApplication) CancelJob(jobID string) (*jobs.Job, error) {
	args := m.Called(jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jobs.Job), args.Error(1)
}

func (m *MockJobServiceForApplication) ListAllJobs() []*jobs.Job {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]*jobs.Job)
}

func (m *MockJobServiceForApplication) ListJobsByStatus(status jobs.JobStatus) []*jobs.Job {
	args := m.Called(status)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]*jobs.Job)
}

func (m *MockJobServiceForApplication) UpdateJobStatus(ctx context.Context, jobID string, status jobs.JobStatus, progress *jobs.JobProgress) error {
	args := m.Called(ctx, jobID, status, progress)
	return args.Error(0)
}

func (m *MockJobServiceForApplication) ListJobsByType(jobType jobs.JobType) []*jobs.Job {
	args := m.Called(jobType)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]*jobs.Job)
}

func (m *MockJobServiceForApplication) UpdateJobProgress(jobID string, stage, description string, percentage, itemsDone, itemsTotal int) error {
	args := m.Called(jobID, stage, description, percentage, itemsDone, itemsTotal)
	return args.Error(0)
}

// UpdateNotifier interface for job notifications
type UpdateNotifier interface {
	NotifyUpdate()
	NotifyJobUpdate(jobID string, job *jobs.Job)
}

func (m *MockJobServiceForApplication) SetUpdateNotifier(notifier UpdateNotifier) {
	m.Called(notifier)
}

// MockSiteContentAggregateRepository implements SiteContentAggregateRepository for testing
type MockSiteContentAggregateRepository struct {
	mock.Mock
}

func (m *MockSiteContentAggregateRepository) GetSiteWithMetadata(ctx context.Context, siteID int64) (*contracts.SiteWithMetadata, error) {
	args := m.Called(ctx, siteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*contracts.SiteWithMetadata), args.Error(1)
}

func (m *MockSiteContentAggregateRepository) GetAllSitesWithMetadata(ctx context.Context) ([]*contracts.SiteWithMetadata, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*contracts.SiteWithMetadata), args.Error(1)
}

func (m *MockSiteContentAggregateRepository) SearchSites(ctx context.Context, searchQuery string) ([]*contracts.SiteWithMetadata, error) {
	args := m.Called(ctx, searchQuery)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*contracts.SiteWithMetadata), args.Error(1)
}

func (m *MockSiteContentAggregateRepository) GetListByID(ctx context.Context, siteID int64, listID string) (*sharepoint.List, error) {
	args := m.Called(ctx, siteID, listID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sharepoint.List), args.Error(1)
}

func (m *MockSiteContentAggregateRepository) GetListsForSite(ctx context.Context, siteID int64) ([]*sharepoint.List, error) {
	args := m.Called(ctx, siteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.List), args.Error(1)
}

func (m *MockSiteContentAggregateRepository) GetListAssignmentsWithRootCause(ctx context.Context, siteID int64, listID string) ([]*sharepoint.ResolvedAssignment, error) {
	args := m.Called(ctx, siteID, listID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.ResolvedAssignment), args.Error(1)
}

func (m *MockSiteContentAggregateRepository) GetAssignmentsForObject(ctx context.Context, siteID int64, objectType, objectKey string) ([]*sharepoint.Assignment, error) {
	args := m.Called(ctx, siteID, objectType, objectKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Assignment), args.Error(1)
}

func (m *MockSiteContentAggregateRepository) GetListItems(ctx context.Context, siteID int64, listID string, offset, limit int) ([]*sharepoint.Item, error) {
	args := m.Called(ctx, siteID, listID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Item), args.Error(1)
}

func (m *MockSiteContentAggregateRepository) GetListSharingLinks(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLink, error) {
	args := m.Called(ctx, siteID, listID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.SharingLink), args.Error(1)
}

func (m *MockSiteContentAggregateRepository) GetListSharingLinksWithItemData(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLinkWithItemData, error) {
	args := m.Called(ctx, siteID, listID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.SharingLinkWithItemData), args.Error(1)
}

func (m *MockSiteContentAggregateRepository) GetSharingLinkMembers(ctx context.Context, siteID int64, linkID string) ([]*sharepoint.Principal, error) {
	args := m.Called(ctx, siteID, linkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Principal), args.Error(1)
}

func (m *MockSiteContentAggregateRepository) GetLastAuditDate(ctx context.Context, siteID int64) (*time.Time, error) {
	args := m.Called(ctx, siteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

// MockAuditRepository implements AuditRepository for testing
type MockAuditRepository struct {
	mock.Mock
}

func (m *MockAuditRepository) SaveSite(ctx context.Context, site *sharepoint.Site) error {
	args := m.Called(ctx, site)
	return args.Error(0)
}

func (m *MockAuditRepository) GetSiteByURL(ctx context.Context, siteURL string) (*sharepoint.Site, error) {
	args := m.Called(ctx, siteURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sharepoint.Site), args.Error(1)
}

func (m *MockAuditRepository) SaveWeb(ctx context.Context, auditRunID int64, web *sharepoint.Web) error {
	args := m.Called(ctx, auditRunID, web)
	return args.Error(0)
}

func (m *MockAuditRepository) SaveList(ctx context.Context, auditRunID int64, list *sharepoint.List) error {
	args := m.Called(ctx, auditRunID, list)
	return args.Error(0)
}

func (m *MockAuditRepository) SaveItem(ctx context.Context, auditRunID int64, item *sharepoint.Item) error {
	args := m.Called(ctx, auditRunID, item)
	return args.Error(0)
}

func (m *MockAuditRepository) SaveRoleDefinitions(ctx context.Context, auditRunID int64, siteID int64, roleDefs []*sharepoint.RoleDefinition) error {
	args := m.Called(ctx, auditRunID, siteID, roleDefs)
	return args.Error(0)
}

func (m *MockAuditRepository) SavePrincipal(ctx context.Context, auditRunID int64, principal *sharepoint.Principal) error {
	args := m.Called(ctx, auditRunID, principal)
	return args.Error(0)
}

func (m *MockAuditRepository) SaveRoleAssignments(ctx context.Context, auditRunID int64, siteID int64, assignments []*sharepoint.RoleAssignment) error {
	args := m.Called(ctx, auditRunID, siteID, assignments)
	return args.Error(0)
}

func (m *MockAuditRepository) ClearRoleAssignments(ctx context.Context, siteID int64, objectType, objectKey string) error {
	args := m.Called(ctx, siteID, objectType, objectKey)
	return args.Error(0)
}

func (m *MockAuditRepository) SaveSharingLinks(ctx context.Context, auditRunID int64, siteID int64, links []*sharepoint.SharingLink) error {
	args := m.Called(ctx, auditRunID, siteID, links)
	return args.Error(0)
}

func (m *MockAuditRepository) ClearSharingLinks(ctx context.Context, siteID int64, itemGUID string) error {
	args := m.Called(ctx, siteID, itemGUID)
	return args.Error(0)
}

func (m *MockAuditRepository) GetAllSharingLinks(ctx context.Context, siteID int64) ([]*sharepoint.Principal, error) {
	args := m.Called(ctx, siteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Principal), args.Error(1)
}

func (m *MockAuditRepository) GetFlexibleSharingLinks(ctx context.Context, siteID int64) ([]*sharepoint.Principal, error) {
	args := m.Called(ctx, siteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Principal), args.Error(1)
}

func (m *MockAuditRepository) GetItemByGUID(ctx context.Context, siteID int64, itemGUID string) (*sharepoint.Item, error) {
	args := m.Called(ctx, siteID, itemGUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sharepoint.Item), args.Error(1)
}

func (m *MockAuditRepository) GetItemByListItemGUID(ctx context.Context, siteID int64, listItemGUID string) (*sharepoint.Item, error) {
	args := m.Called(ctx, siteID, listItemGUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sharepoint.Item), args.Error(1)
}

func (m *MockAuditRepository) GetItemByListAndID(ctx context.Context, siteID int64, listID string, itemID int64) (*sharepoint.Item, error) {
	args := m.Called(ctx, siteID, listID, itemID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sharepoint.Item), args.Error(1)
}

func (m *MockAuditRepository) SaveItemSensitivityLabel(ctx context.Context, label *sharepoint.ItemSensitivityLabel) error {
	args := m.Called(ctx, label)
	return args.Error(0)
}

func (m *MockAuditRepository) SaveSharingGovernance(ctx context.Context, siteID int64, sharingInfo *sharepoint.SharingInfo) error {
	args := m.Called(ctx, siteID, sharingInfo)
	return args.Error(0)
}

func (m *MockAuditRepository) SaveSharingAbilities(ctx context.Context, siteID int64, abilities *sharepoint.SharingAbilities) error {
	args := m.Called(ctx, siteID, abilities)
	return args.Error(0)
}

func (m *MockAuditRepository) SaveRecipientLimits(ctx context.Context, siteID int64, limits *sharepoint.RecipientLimits) error {
	args := m.Called(ctx, siteID, limits)
	return args.Error(0)
}

func (m *MockAuditRepository) SaveSensitivityLabel(ctx context.Context, siteID int64, itemGUID string, label *sharepoint.SensitivityLabelInformation) error {
	args := m.Called(ctx, siteID, itemGUID, label)
	return args.Error(0)
}

// Audit-aware query operations
func (m *MockAuditRepository) GetSitesByAuditRun(ctx context.Context, auditRunID int64) ([]*sharepoint.Site, error) {
	args := m.Called(ctx, auditRunID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Site), args.Error(1)
}

func (m *MockAuditRepository) GetWebsByAuditRun(ctx context.Context, auditRunID int64) ([]*sharepoint.Web, error) {
	args := m.Called(ctx, auditRunID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Web), args.Error(1)
}

func (m *MockAuditRepository) GetListsByAuditRun(ctx context.Context, auditRunID int64) ([]*sharepoint.List, error) {
	args := m.Called(ctx, auditRunID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.List), args.Error(1)
}

func (m *MockAuditRepository) GetItemsByAuditRun(ctx context.Context, auditRunID int64) ([]*sharepoint.Item, error) {
	args := m.Called(ctx, auditRunID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sharepoint.Item), args.Error(1)
}

// Ensure mocks implement the interfaces at compile time
var _ contracts.SiteRepository = (*MockSiteRepository)(nil)
var _ contracts.ListRepository = (*MockListRepository)(nil)
var _ contracts.JobRepository = (*MockJobRepository)(nil)
var _ contracts.AssignmentRepository = (*MockAssignmentRepository)(nil)
var _ contracts.ItemRepository = (*MockItemRepository)(nil)
var _ contracts.SharingRepository = (*MockSharingRepository)(nil)
var _ contracts.AuditRepository = (*MockAuditRepository)(nil)
var _ contracts.SiteContentAggregateRepository = (*MockSiteContentAggregateRepository)(nil)
