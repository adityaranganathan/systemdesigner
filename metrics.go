package main

import "math"

type Message struct {
	NodeID  string   `json:"nodeId"`
	Metrics []Metric `json:"metrics"`
}

type Metric struct {
	Name     string  `json:"name"`
	Value    int     `json:"value"`
	Unit     string  `json:"unit"`
	Severity float64 `json:"severity"`
}

func NewNumResponses(value int) Metric {
	return Metric{
		Name:  "Responses",
		Value: value,
		Unit:  "",
	}
}

func NewAvgLatency(value int) Metric {
	var severity float64

	if value <= 500 {
		severity = 0
	} else if value >= 1000 {
		severity = 1
	} else {
		severity = float64(value-500) / 500.0
	}

	return Metric{
		Name:     "Avg. Latency",
		Value:    value,
		Unit:     "ms",
		Severity: severity,
	}
}

func NewQueued(value int) Metric {
	severity := math.Min(1, float64(value)/100.0)
	return Metric{
		Name:     "Queued",
		Value:    value,
		Unit:     "reqs",
		Severity: severity,
	}
}

func NewUtilisation(value int) Metric {
	return Metric{
		Name:     "Utilisation",
		Value:    value,
		Unit:     "%",
		Severity: float64(value) / 100,
	}
}

func NewProcessed(value int) Metric {
	return Metric{
		Name:  "Processed",
		Value: value,
		Unit:  "reqs",
	}
}
