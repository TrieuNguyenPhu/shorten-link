package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"
	"time"
)

func EmitCountMetric(w io.Writer, namespace, service, metricName string, value float64) error {
	if strings.TrimSpace(namespace) == "" || len(namespace) > 255 || strings.TrimSpace(service) == "" || strings.TrimSpace(metricName) == "" || len(metricName) > 255 || metricName == "service" || metricName == "_aws" || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return errors.New("invalid count metric")
	}
	return json.NewEncoder(w).Encode(map[string]any{
		"_aws": map[string]any{
			"Timestamp": time.Now().UnixMilli(),
			"CloudWatchMetrics": []any{map[string]any{
				"Namespace":  namespace,
				"Dimensions": [][]string{{"service"}},
				"Metrics":    []any{map[string]string{"Name": metricName, "Unit": "Count"}},
			}},
		},
		"service":  service,
		metricName: value,
	})
}
