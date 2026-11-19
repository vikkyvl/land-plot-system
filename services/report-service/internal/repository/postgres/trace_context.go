package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/tracing"
)

func currentTraceContext(source context.Context) ([]byte, error) {
	traceContext := tracing.Inject(source)
	if len(traceContext) == 0 {
		return nil, nil
	}

	encoded, err := json.Marshal(traceContext)
	if err != nil {
		return nil, fmt.Errorf(wrapEncodeTraceContext, err)
	}

	return encoded, nil
}

func decodeTraceContext(encoded []byte) (domain.TraceContext, error) {
	if len(encoded) == 0 {
		return nil, nil
	}

	var traceContext domain.TraceContext
	if err := json.Unmarshal(encoded, &traceContext); err != nil {
		return nil, fmt.Errorf(wrapDecodeTraceContext, err)
	}

	return traceContext, nil
}
