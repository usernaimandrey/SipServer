package sipserver

import (
	"time"

	"github.com/emiago/sipgo/sip"
)

type DialogCtx struct {
	Key          string
	RouteSet     []*sip.RouteHeader
	RemoteTarget sip.Uri
	JournalID    int64
	CallID       string
	FromTag      string
	ToTag        string
	CallerUser   string
	CalleeUser   string
	AnswerAt     time.Time
}
