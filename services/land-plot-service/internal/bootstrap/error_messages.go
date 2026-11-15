package bootstrap

const (
	wrapDetermineServiceAddress = "determine service address: %w"
	wrapListenHTTP              = "listen HTTP: %w"
	wrapLoadConfig              = "load config: %w"
	wrapMigrateDatabase         = "migrate database: %w"
	wrapOpenDatabasePool        = "open database pool: %w"
	wrapPingDatabase            = "ping database: %w"
	wrapRegisterInConsul        = "register in consul: %w"
	wrapServeHTTP               = "serve HTTP: %w"
	wrapShutdownHTTPServer      = "shutdown HTTP server: %w"
)
