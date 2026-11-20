package postgres

import (
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

var ErrSimulatedWriteFailure = fmt.Errorf(wrapSimulatedWriteFailureCause, domain.ErrRejectedEvent)
