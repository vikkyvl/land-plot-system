package service

const (
	logClimateUnavailable             = "climate unavailable for report"
	logTerrainUnavailable             = "terrain unavailable for report"
	logReadClimateCache               = "read climate cache"
	logWriteClimateCache              = "write climate cache"
	logDeletePublishedOutboxMessages  = "delete published outbox messages"
	logOutboxMessageAbandoned         = "outbox message abandoned after too many rejections"
	logOutboxMessagesPublished        = "outbox messages published"
	logPublishedOutboxMessagesDeleted = "published outbox messages deleted"
	logRelayOutboxMessages            = "relay outbox messages"
)
