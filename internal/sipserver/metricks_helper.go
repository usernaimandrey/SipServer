package sipserver

import (
	"strconv"
	"time"

	"SipServer/internal/metrics"

	"github.com/emiago/sipgo/sip"
)

func sipIn(method sip.RequestMethod) {
	metrics.SIPMessages.WithLabelValues(string(method), "IN").Inc()
}

func sipOut(method sip.RequestMethod) {
	metrics.SIPMessages.WithLabelValues(string(method), "OUT").Inc()
}

func sipActiveDialogsSet(count int64) {
	metrics.SIPActiveDialogs.Set(float64(count))
}

func sipResp(method sip.RequestMethod, code int) {
	metrics.SIPResponses.WithLabelValues(string(method), strconv.Itoa(code)).Inc()
}

func respond(req *sip.Request, tx sip.ServerTransaction, code int, reason string, headers ...sip.Header) (*sip.Response, error) {
	resp := sip.NewResponseFromRequest(req, code, reason, nil)
	for _, h := range headers {
		resp.AppendHeader(h)
	}
	sipResp(req.Method, int(code))
	sipOut(req.Method)
	return resp, tx.Respond(resp)
}

func observeHandler(method sip.RequestMethod, start time.Time) {
	metrics.SIPHandlerDuration.WithLabelValues(string(method)).Observe(time.Since(start).Seconds())
}
