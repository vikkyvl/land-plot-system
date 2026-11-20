package tools

const (
	ListPlots         = "list_plots"
	GetPlot           = "get_plot"
	CreatePlot        = "create_plot"
	UpdatePlot        = "update_plot"
	WaitPlotConfirmed = "wait_plot_confirmed"
	CreateReport      = "create_report"
	GetReport         = "get_report"
)

var (
	AllNames      = []string{ListPlots, GetPlot, CreatePlot, UpdatePlot, WaitPlotConfirmed, CreateReport, GetReport}
	ProducerNames = []string{ListPlots, GetPlot, CreatePlot, UpdatePlot, WaitPlotConfirmed}
	ConsumerNames = []string{GetPlot, CreateReport, GetReport}
)

const (
	OutcomeConfirmed    = "confirmed"
	OutcomeRolledBack   = "rolled_back"
	OutcomeCancelled    = "cancelled"
	OutcomeStillPending = "still_pending"
)
