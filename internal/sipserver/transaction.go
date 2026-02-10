package sipserver

import (
	"sync/atomic"
	"time"

	"github.com/emiago/sipgo/sip"
)

type InviteCtx struct {
	Key           string
	ServerTx      sip.ServerTransaction
	LastResp      *sip.Response
	OriginInvite  *sip.Request
	CreatedAt     time.Time
	ClientTx      sip.ClientTransaction
	DialogCreated atomic.Bool
	OutInvite     *sip.Request
	FinalRespCode int
	Got2xx        bool
	JournalID     int64
	InviteAt      time.Time
}

func NewInviteCtx() *InviteCtx {
	return &InviteCtx{}
}

func inviteKeyFromReq(req *sip.Request) (string, bool) {
	via := req.Via()
	if via == nil {
		return "", false
	}
	branch, ok := via.Params.Get("branch")
	if !ok || branch == "" {
		return "", false
	}
	return "INVITE|" + branch, true
}
