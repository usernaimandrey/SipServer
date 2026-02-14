package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HTTPInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_in_flight_requests",
		Help: "Current number of in-flight HTTP requests.",
	})

	HTTPRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "route", "status"})

	HTTPDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latencies in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})

	SIPMessages = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "sip_messages_total",
		Help: "Total number of SIP messages.",
	}, []string{"method", "direction"}) // IN/OUT

	SIPResponses = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "sip_responses_total",
		Help: "Total number of SIP responses.",
	}, []string{"method", "code"})

	SIPHandlerDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sip_handler_duration_seconds",
		Help:    "SIP handler duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method"})

	SIPActiveDialogs = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sip_active_dialogs",
		Help: "Number of active SIP dialogs tracked by server.",
	})

	SIPRegistrations = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sip_registrations",
		Help: "Number of active registrations in registrar.",
	})

	SIPTransactionsInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sip_transactions_in_flight",
		Help: "Number of in-flight INVITE transactions stored in server.transaction.",
	})

	SIPDialogEntries = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sip_dialog_entries",
		Help: "Number of dialog entries stored in server.dialogs (usually 2 per dialog).",
	})
)

func MustRegister(reg prometheus.Registerer) {
	reg.MustRegister(
		HTTPInFlight, HTTPRequests, HTTPDuration,
		SIPMessages, SIPResponses, SIPHandlerDuration,
		SIPActiveDialogs, SIPRegistrations, SIPTransactionsInFlight,
		SIPDialogEntries,
	)
}
