package sipserver

import (
	"strings"

	"github.com/emiago/sipgo/sip"
)

func setRequestURIAndDest(req *sip.Request, uri sip.Uri) {

	req.Recipient = uri

	req.SetDestination(uri.HostPort())
}

func decreaseMaxForwards(req *sip.Request) {
	mf := req.MaxForwards()
	if mf == nil {
		h := sip.MaxForwardsHeader(70)
		req.AppendHeader(&h)
		return
	}
	mf.Dec()
}

func makeInviteKey(req *sip.Request) inviteKey {
	callID := ""
	if req.CallID() != nil {
		callID = req.CallID().String()
	}

	cseqNo := ""
	if req.CSeq() != nil {
		parts := strings.SplitN(req.CSeq().String(), " ", 2)
		cseqNo = strings.TrimSpace(parts[0])
	}

	return inviteKey{callID: callID, cseqNo: cseqNo}
}

func stripTopVia(msg interface {
	GetHeaders(name string) []sip.Header
	RemoveHeader(name string) bool
	PrependHeader(headers ...sip.Header)
}) {
	vias := msg.GetHeaders("via")
	if len(vias) == 0 {
		return
	}

	msg.RemoveHeader("Via")

	for i := len(vias) - 1; i >= 1; i-- {
		msg.PrependHeader(vias[i])
	}
}
