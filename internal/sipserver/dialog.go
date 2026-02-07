package sipserver

import (
	"github.com/emiago/sipgo/sip"
)

type DialogCtx struct {
	Key          string
	RouteSet     []*sip.RouteHeader
	RemoteTarget sip.Uri
}
