package domain

type PlotSummary struct {
	Count       int
	TotalAreaM2 float64
}

func SummarizePlots(plots []Plot) PlotSummary {
	summary := PlotSummary{Count: len(plots)}
	for _, plot := range plots {
		summary.TotalAreaM2 += plot.AreaM2
	}

	return summary
}
