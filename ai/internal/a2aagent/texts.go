package a2aagent

const (
	producerName        = "Land plot producer agent"
	producerDescription = "AI agent over land-plot-service, the event producer of the system. It creates or updates " +
		"a land plot, which starts a saga and produces plot.created or plot.updated, and waits until the saga " +
		"is confirmed or compensated."

	skillCreatePlotDescription = "Create a land plot from a name and polygon points and wait for the saga. " +
		"Returns an artifact with plot_id, status and version; the task fails if the saga is compensated."
	skillCreatePlotExample     = `Створи ділянку "Пай під Білою Церквою" з точками (49.7968, 30.1012), (49.7968, 30.1105), (49.8021, 30.1105), (49.8021, 30.1012)`
	skillUpdatePlotDescription = "Rename a land plot or replace its polygon and wait for the saga. " +
		"The task fails if the update is rolled back."
	skillUpdatePlotExample = `Перейменуй ділянку "Дача в Ірпені" на "Дача в Ірпені, вул. Садова 12"`

	taskDataPrefix       = "Дані задачі (JSON): "
	formatSagaOutcome    = "Сага завершилась без підтвердження: %s"
	reasonSagaNotAwaited = "Сага запущена, але її завершення не дочекались"
	reasonNoPlot         = "Ділянку не створено і не змінено"

	logTaskFinished     = "A2A task finished"
	logDelegated        = "task delegated to the producer agent"
	logDelegationFailed = "delegation to the producer agent failed"

	consumerName        = "Land plot consumer agent"
	consumerDescription = "AI agent over report-service, the event consumer of the system. It builds area reports " +
		"and delegates creation of missing plots to the producer agent through A2A."
	skillPlotReportDescription = "Build an area report for a plot; if the plot does not exist yet, ask the producer " +
		"agent to create it first. Returns an artifact with report_id, plot_id and area in hectares."
	skillPlotReportExample = `Створи ділянку "Пай під Білою Церквою" з точками (49.7968, 30.1012), (49.7968, 30.1105), ` +
		`(49.8021, 30.1105), (49.8021, 30.1012) і сформуй звіт`

	descriptionDelegate = "Send a task to the trusted producer agent through A2A and wait for its result. " +
		"Use skill create_plot to create a plot or update_plot to change one. The result has producer_state " +
		"(completed or failed), plot with plot_id, status and version, and the producer message."
	descriptionDelegateSkill   = "Producer skill: create_plot or update_plot."
	descriptionDelegateRequest = "Full task text for the producer with the plot name and all points as (latitude, longitude)."

	reasonProducerFailed          = "Producer-agent не виконав задачу"
	reasonNoReportAfterDelegation = "Ділянку делеговано producer-agent, але звіт не сформовано"
	reasonNoReport                = "Звіт не сформовано"

	codeInvalidArguments    = "invalid_arguments"
	codeProducerUnavailable = "producer_unavailable"

	formatMissingSkill = "producer agent card has no skill %s"
	formatUnknownSkill = "skill must be create_plot or update_plot, got %q"
	wrapResolveCard    = "read producer agent card at %s: %w"
	wrapCreateClient   = "create A2A client: %w"
	wrapSendTask       = "send task to the producer agent: %w"
	wrapDecodeDelegate = "arguments must be a JSON object with skill and request only: %w"
)
