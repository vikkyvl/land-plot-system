package postgres

const (
	wrapApplyReportSchema           = "apply report schema: %w"
	wrapBeginOutboxTransaction      = "begin outbox transaction: %w"
	wrapBeginPlotEventTransaction   = "begin plot event transaction: %w"
	wrapClaimOutboxEvents           = "claim outbox events: %w"
	wrapCommitOutboxTransaction     = "commit outbox transaction: %w"
	wrapCommitPlotEvent             = "commit plot event: %w"
	wrapDeletePublishedOutboxEvents = "delete published outbox events: %w"
	wrapEncodeSummaryApplied        = "encode summary applied event: %w"
	wrapInsertOutboxEvent           = "insert outbox event: %w"
	wrapInsertReport                = "insert report: %w"
	wrapMarkOutboxEventPublished    = "mark outbox event published: %w"
	wrapPublishOutboxEvent          = "publish outbox event: %w"
	wrapQueryReportByID             = "query report by ID: %w"
	wrapRecordOutboxFailure         = "record outbox failure: %w"
	wrapRecordProcessedEvent        = "record processed event: %w"
	wrapSimulatedWriteFailure       = "%w for plot %q"
	wrapSimulatedWriteFailureCause  = "%w: simulated plot summary write failure"
	wrapUpsertPlotSummary           = "upsert plot summary: %w"
)
