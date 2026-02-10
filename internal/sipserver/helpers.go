package sipserver

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/emiago/sipgo/sip"
)

func decreaseMaxForwards(req *sip.Request) error {
	mf := req.MaxForwards()
	if mf == nil {
		h := sip.MaxForwardsHeader(70)
		req.AppendHeader(&h)
		return nil
	}

	if mf.Val() == 0 {
		return errors.New("to many hops")
	}
	mf.Dec()

	return nil
}

func hasViaLoop(req *sip.Request, hostport string) error {

	vias := req.GetHeaders("Via")
	for _, h := range vias {
		vh, ok := h.(*sip.ViaHeader)
		if !ok || vh == nil {
			continue
		}

		if strings.EqualFold(vh.SentBy(), hostport) {
			return errors.New("loop detected")
		}

	}
	return nil
}

func buildOutboundInvite(in *sip.Request, target *sip.Uri, myHost string, myPort int) *sip.Request {
	out := sip.NewRequest(sip.INVITE, *target)

	copyFrom := *in.From()
	copyTo := *in.To()
	copyCallId := *in.CallID()
	copyCSeq := *in.CSeq()

	out.AppendHeader(&copyFrom)
	out.AppendHeader(&copyTo)
	out.AppendHeader(&copyCallId)
	out.AppendHeader(&copyCSeq)

	if h := in.Contact(); h != nil {
		out.AppendHeader(h.Clone())
	}

	var mf uint32 = 70
	if h := in.MaxForwards(); h != nil {
		mf = h.Val()
	}
	if mf <= 0 {
		mf = 0
	} else {
		mf--
	}

	var maxForwardsHeader sip.MaxForwardsHeader = sip.MaxForwardsHeader(mf)
	out.AppendHeader(&maxForwardsHeader)

	rr := &sip.RecordRouteHeader{
		Address: sip.Uri{
			Scheme:    "sip",
			Host:      myHost,
			Port:      myPort,
			UriParams: sip.NewParams().Add("lr", ""),
		},
	}
	out.AppendHeader(rr)

	for _, h := range in.GetHeaders("Via") {
		if vh, ok := h.(*sip.ViaHeader); ok && vh != nil {
			c := *vh
			out.AppendHeader(&c)
			continue
		}
		out.AppendHeader(h)
	}

	via := &sip.ViaHeader{
		Transport: "UDP",
		Host:      myHost,
		Port:      myPort,
		Params:    sip.NewParams(),
	}
	via.Params.Add("branch", sip.GenerateBranch())
	via.Params.Add("rport", "")
	via.ProtocolName = "SIP"
	via.ProtocolVersion = "2.0"
	out.PrependHeader(via)

	if body := in.Body(); len(body) > 0 {
		out.SetBody(body)
		if ct := in.ContentType(); ct != nil {
			cloneCt := *ct
			out.AppendHeader(&cloneCt)
		} else {
			var contentType sip.ContentTypeHeader = sip.ContentTypeHeader("application/sdp")
			out.AppendHeader(&contentType)
		}
	}

	return out
}

func makeUpstreamResponse(orig *sip.Request, down *sip.Response) *sip.Response {
	up := sip.NewResponseFromRequest(orig, down.StatusCode, down.Reason, nil)

	up.RemoveHeader("Via")
	up.RemoveHeader("Contact")
	up.RemoveHeader("Content-Type")
	up.RemoveHeader("Content-Length")
	up.RemoveHeader("Record-Route")

	// 2) Via: для ответа upstream достаточно VIA из orig (в правильном порядке)
	for _, h := range orig.GetHeaders("Via") {
		if vh, ok := h.(*sip.ViaHeader); ok && vh != nil {
			c := *vh
			up.AppendHeader(&c)
		} else {
			up.AppendHeader(h)
		}
	}

	copyFrom := *down.From()
	copyTo := *down.To()
	copyCallId := *down.CallID()
	copyCSeq := *down.CSeq()

	up.ReplaceHeader(&copyFrom)
	up.ReplaceHeader(&copyTo)
	up.ReplaceHeader(&copyCallId)
	up.ReplaceHeader(&copyCSeq)

	for _, h := range down.GetHeaders("Record-Route") {
		up.AppendHeader(h) // можно Clone если нужно
	}
	if c := down.Contact(); c != nil {
		up.AppendHeader(c.Clone())
	}

	up.RemoveHeader("Content-Length")
	up.RemoveHeader("Content-Type")

	// 5) Body + CT + CL — делаем Replace (один экземпляр)
	body := down.Body()
	if len(body) > 0 {
		up.SetBody(body)

		if ct := down.ContentType(); ct != nil {
			clone := *ct
			up.AppendHeader(&clone)
		} else {
			contentType := sip.ContentTypeHeader("application/sdp")
			up.AppendHeader(&contentType)
		}

		cl := sip.ContentLengthHeader(len(body))
		up.AppendHeader(&cl)
	} else {
		cl := sip.ContentLengthHeader(0)
		up.AppendHeader(&cl)
	}

	return up
}

func MakeDialogKey(callID, tagA, tagB string) (string, string) {
	key1 := callID + "|" + tagA + "|" + tagB
	key2 := callID + "|" + tagB + "|" + tagA

	return key1, key2
}

func buildRouteSet(resp *sip.Response) []*sip.RouteHeader {
	var rr []*sip.RecordRouteHeader
	for _, h := range resp.GetHeaders("Record-Route") {
		if x, ok := h.(*sip.RecordRouteHeader); ok && x != nil {
			rr = append(rr, x)
		}
	}

	routes := make([]*sip.RouteHeader, 0, len(rr))
	for i := len(rr) - 1; i >= 0; i-- {
		routes = append(routes, &sip.RouteHeader{Address: rr[i].Address})
	}
	return routes
}

func stripSelfRoute(routes []*sip.RouteHeader, myHost string, myPort int) []*sip.RouteHeader {
	out := routes
	for len(out) > 0 {
		r := out[0]
		if r != nil && r.Address.Host == myHost && int(r.Address.Port) == myPort {
			out = out[1:]
			continue
		}
		break
	}
	return out
}

func extractTopViaBranch(req *sip.Request) string {
	vias := req.GetHeaders("Via")
	if vias == nil || len(vias) == 0 {
		return ""
	}

	vh, ok := vias[0].(*sip.ViaHeader)

	if !ok {
		return ""
	}

	b, _ := vh.Params.Get("branch")
	return b
}

func encodeRouteSet(routes []*sip.RouteHeader) ([]byte, error) {
	if len(routes) == 0 {
		return []byte("[]"), nil
	}

	out := make([]string, 0, len(routes))
	for _, r := range routes {
		out = append(out, r.Address.String())
	}

	return json.Marshal(out)
}
