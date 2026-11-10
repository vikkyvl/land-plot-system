package service

const (
	logCompensateExpiredSaga          = "compensate expired saga"
	logCompensateExpiredSagas         = "compensate expired sagas"
	logDeletePublishedOutboxMessages  = "delete published outbox messages"
	logOutboxMessageAbandoned         = "outbox message abandoned after too many rejections"
	logOutboxMessagesPublished        = "outbox messages published"
	logPublishedOutboxMessagesDeleted = "published outbox messages deleted"
	logRelayOutboxMessages            = "relay outbox messages"
	logSagaCompensated                = "plot saga compensated"
	logSagaCompleted                  = "plot saga completed"
)
