package view

import (
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestHausv615EveryAuditActionHasGermanLabel(t *testing.T) {
	actions := []string{
		store.AuditActionHandoverCreate, store.AuditActionHandoverConfirm, store.AuditActionHandoverFile,
		store.AuditActionLogin, store.AuditActionContextSwitch,
		store.AuditActionInviteCreate, store.AuditActionInviteUpdate, store.AuditActionInviteDelete,
		store.AuditActionBuildingUpdate, store.AuditActionPortalModulesUpdate, store.AuditActionHeroUpdate,
		store.AuditActionUnitSave, store.AuditActionUnitDelete, store.AuditActionUnitPayment,
		store.AuditActionLeaseCreate, store.AuditActionLeaseUpdate, store.AuditActionLeaseEnd,
		store.AuditActionLeasePartyChange, store.AuditActionRentComponentAdd,
		store.AuditActionClauseCreate, store.AuditActionClauseUpdate, store.AuditActionClauseReview,
		store.AuditActionAnnualPeriodSave, store.AuditActionAnnualPartiesImport, store.AuditActionAnnualCostTypeSave,
		store.AuditActionAnnualBasesSave, store.AuditActionAnnualReceiptCreate, store.AuditActionAnnualReceiptAmount,
		store.AuditActionAnnualReceiptDelete, store.AuditActionAnnualPrepaymentSave, store.AuditActionAnnualReserveAdd, store.AuditActionAnnualRunCreate,
		store.AuditActionParkingSettings, store.AuditActionParkingMonth, store.AuditActionParkingReminder,
		store.AuditActionChargingSettings, store.AuditActionChargingManual, store.AuditActionChargingSession,
		store.AuditActionIssueWorkflow, store.AuditActionIssueEstimate, store.AuditActionIssueServiceAdd,
		store.AuditActionIssueServiceDrop, store.AuditActionIssueComment, store.AuditActionIssueCommentDelete,
		store.AuditActionEventCreate, store.AuditActionEventUpdate, store.AuditActionEventDelete,
		store.AuditActionContactSave, store.AuditActionContactDelete,
		store.AuditActionDocumentUpload, store.AuditActionDocumentDownload, store.AuditActionDocumentReplace,
		store.AuditActionAttachmentView, store.AuditActionAttachmentDelete,
		store.AuditActionIntegrationImport, store.AuditActionIntegrationExport,
		store.AuditActionVoteCreate, store.AuditActionVoteOpen, store.AuditActionVoteClose,
		store.AuditActionVoteCast, store.AuditActionVoteReminder,
		store.AuditActionEnergyOnboarding, store.AuditActionEnergyIdentity, store.AuditActionEnergyMode,
		store.AuditActionEnergyImport, store.AuditActionEnergyTarget, store.AuditActionEnergyRecommend,
		store.AuditActionEnergyMeasureAdd, store.AuditActionEnergyMeasureEdit, store.AuditActionEnergyCaretaker,
		store.AuditActionEnergyInvite, store.AuditActionEnergyMaintSave, store.AuditActionEnergyMaintDone,
		store.AuditActionEnergyTariff, store.AuditActionEnergyExport, store.AuditActionEnergyHistoryDelete,
		store.AuditActionEnergyProfileDelete,
		store.AuditActionIssueAISuggest, store.AuditActionIssueAIAccept, store.AuditActionIssueAIEdit,
		store.AuditActionIssueAIReject, store.AuditActionIssueAIAuto, store.AuditActionIssueAIRestore,
		store.AuditActionIntakePhoneNote, store.AuditActionIntakeAssign,
		store.AuditActionVerwaltungSettings, store.AuditActionDemoReset, store.AuditActionTextbausteinChanged,
		store.AuditActionRolePreviewStart, store.AuditActionRolePreviewEnd,
	}
	for _, action := range actions {
		label := AuditActionLabel(action)
		if label == "" || label == "Aktivität" || label == action || strings.Contains(label, ".") {
			t.Errorf("AuditActionLabel(%q) = %q", action, label)
		}
	}
	if got := AuditActionLabel("future.audit.action"); got != "Aktivität" {
		t.Fatalf("unknown audit fallback = %q", got)
	}
}
