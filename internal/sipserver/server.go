package sipserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"

	"SipServer/internal/entity/user"
	"SipServer/internal/registrar"

	"github.com/joho/godotenv"
)

var userNotFound *user.UserNotFoundError

type Server struct {
	srv         *sipgo.Server
	cl          *sipgo.Client
	reg         *registrar.Registrar
	db          *sql.DB
	hostport    string
	host        string
	port        int
	transaction sync.Map
	dialogs     sync.Map
}

func New(ua *sipgo.UserAgent, reg *registrar.Registrar, db *sql.DB) (*Server, error) {
	srv, err := sipgo.NewServer(ua)
	if err != nil {
		return nil, err
	}

	host := os.Getenv("HOST")
	port := os.Getenv("PORT")

	if port == "" {
		port = "5060"
	}

	portInt, err := strconv.Atoi(port)

	if err != nil {
		return nil, err
	}

	cl, err := sipgo.NewClient(
		ua,
		sipgo.WithClientHostname(host),
		sipgo.WithClientPort(portInt),
	)

	err = godotenv.Load()

	if err != nil {
		return nil, err
	}

	s := &Server{
		srv:         srv,
		cl:          cl,
		reg:         reg,
		db:          db,
		hostport:    fmt.Sprintf("%s:%s", host, port),
		host:        host,
		port:        portInt,
		transaction: sync.Map{},
		dialogs:     sync.Map{},
	}

	// REGISTER / INVITE / BYE — ключевые методы для прототипа
	srv.OnRegister(s.onRegister) // хендлеры вида func(req *sip.Request, tx sip.ServerTransaction) :contentReference[oaicite:1]{index=1}
	srv.OnInvite(s.onInvite)
	srv.OnBye(s.onBye)
	srv.OnAck(s.onAck)

	// На всякий случай: если прилетит что-то ещё
	srv.OnNoRoute(func(req *sip.Request, tx sip.ServerTransaction) {
		res := sip.NewResponseFromRequest(req, sip.StatusMethodNotAllowed, "Method Not Allowed", nil)
		_ = tx.Respond(res)
	})

	return s, nil
}

func (s *Server) ListenAndServe(ctx context.Context, network, addr string) error {
	return s.srv.ListenAndServe(ctx, network, addr)
}

func (s *Server) onRegister(req *sip.Request, tx sip.ServerTransaction) {
	// ВНИМАНИЕ: для ВКР-прототипа делаем без Digest-авторизации.
	// В перспективе: 401 + WWW-Authenticate и проверка Authorization.

	from := req.From()
	contact := req.Contact() // Contact() есть у Request :contentReference[oaicite:2]{index=2}

	if from == nil || contact == nil {
		res := sip.NewResponseFromRequest(req, sip.StatusBadRequest, "Bad Request", nil)
		_ = tx.Respond(res)
		return
	}

	login := strings.TrimSpace(from.Address.User)
	if login == "" {
		res := sip.NewResponseFromRequest(req, sip.StatusBadRequest, "Bad Request", nil)
		_ = tx.Respond(res)
		return
	}

	newUser := user.NewUser()
	_, err := newUser.FindByLogin(login, s.db)

	if err != nil {
		if errors.As(err, &userNotFound) {
			log.Printf("user %s not founf", login)
			res := sip.NewResponseFromRequest(req, sip.StatusNotFound, "NotFound", nil)
			_ = tx.Respond(res)
			return
		} else {
			res := sip.NewResponseFromRequest(req, sip.StatusInternalServerError, "InternalError", nil)
			_ = tx.Respond(res)
			return
		}
	}

	src := req.Source()

	reachable, ok := makeReachableContact(login, src)

	if ok {
		s.reg.Put(login, reachable, src, 60*time.Second)
		log.Printf("[REGISTER] user=%s contact=%s (normalized) source=%s",
			login, reachable.String(), src)
	} else {
		// fallback: как есть, но лучше логировать как "подозрительно"
		s.reg.Put(login, contact.Address, src, 60*time.Second)
		log.Printf("[REGISTER] user=%s contact=%s (raw) source=%s",
			login, contact.Address.String(), src)
	}

	// Параметр expires: в SIP бывает и в заголовке Expires, и в Contact; для прототипа возьмём дефолт.
	s.reg.Put(login, contact.Address, req.Source(), 60*time.Second)

	log.Printf("[REGISTER] user=%s contact=%s source=%s", login, contact.Address.String(), req.Source())

	res := sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil)
	_ = tx.Respond(res)
}

func (s *Server) onAck(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	fromTag, _ := req.From().Params.Get("tag")
	toTag, _ := req.To().Params.Get("tag")

	key1, key2 := MakeDialogKey(callID, fromTag, toTag)

	log.Printf("[ACK] Dialog Key: %s  %s", key1, key2)
	v, ok := s.dialogs.Load(key1)
	if !ok {
		v, ok = s.dialogs.Load(key1)
		if !ok {
			log.Printf("[ACK] dialog not found callid=%s fromTag=%s toTag=%s", callID, fromTag, toTag)
			return
		}
	}
	dlg := v.(*DialogCtx)

	ack := sip.NewRequest(sip.ACK, dlg.RemoteTarget)

	log.Println("[ACK] From %v Request target %v", dlg.RemoteTarget, req)

	copyFrom := *req.From()
	copyTo := *req.To()
	copyCallId := *req.CallID()
	copyCSeq := *req.CSeq()

	// копируем From/To/Call-ID/CSeq(ACK) из входящего ACK
	ack.AppendHeader(&copyFrom)
	ack.AppendHeader(&copyTo)
	ack.AppendHeader(&copyCallId)
	ack.AppendHeader(&copyCSeq)

	// Route set
	routes := stripSelfRoute(dlg.RouteSet, s.host, s.port)
	for _, r := range routes {
		ack.AppendHeader(r)
	}

	// Max-Forwards
	var mf sip.MaxForwardsHeader = 70
	ack.AppendHeader(&mf)

	// Via top (свой)
	via := &sip.ViaHeader{
		Transport: "UDP",
		Host:      s.host,
		Port:      s.port,
		Params:    sip.NewParams(),
	}
	via.Params.Add("branch", sip.GenerateBranch())
	via.Params.Add("rport", "")
	via.ProtocolName = "SIP"
	via.ProtocolVersion = "2.0"
	ack.PrependHeader(via)

	// отправляем
	err := s.cl.WriteRequest(ack)
	if err != nil {
		log.Printf("[ACK] forward error: %v", err)
	}
}

func (s *Server) onInvite(req *sip.Request, tx sip.ServerTransaction) {
	to := req.To()
	if to == nil || strings.TrimSpace(to.Address.User) == "" {
		log.Print("Bad Request")
		res := sip.NewResponseFromRequest(req, sip.StatusBadRequest, "Bad Request", nil)
		tx.Respond(res)
		return
	}

	if err := decreaseMaxForwards(req); err != nil {
		log.Print(err.Error())
		res := sip.NewResponseFromRequest(req, sip.StatusTooManyHops, err.Error(), nil)
		tx.Respond(res)
		return
	}

	if err := hasViaLoop(req, s.hostport); err != nil {
		log.Print(err.Error())
		res := sip.NewResponseFromRequest(req, sip.StatusLoopDetected, err.Error(), nil)
		tx.Respond(res)
		return
	}

	key, ok := inviteKeyFromReq(req)

	if !ok {
		tx.Respond(sip.NewResponseFromRequest(req, sip.StatusBadRequest, "Bad Requsets", nil))
		return
	}

	if v, exists := s.transaction.Load(key); exists {
		ctx, ok := v.(*InviteCtx)

		if !ok {
			tx.Respond(sip.NewResponseFromRequest(req, sip.StatusInternalServerError, "Internal Error", nil))
			return
		}

		if ctx.LastResp != nil {
			ctx.ServerTx.Respond(ctx.LastResp)
		} else {
			ctx.ServerTx.Respond(sip.NewResponseFromRequest(req, 100, "Trying", nil))
		}
		return
	}

	newCtx := NewInviteCtx()
	newCtx.OriginInvite = req
	newCtx.ServerTx = tx
	s.transaction.Store(key, newCtx)

	resp100 := sip.NewResponseFromRequest(req, sip.StatusTrying, "Trying", nil)
	newCtx.LastResp = resp100

	tx.Respond(resp100)

	callee := strings.TrimSpace(to.Address.User)

	newUser := user.NewUser()

	_, err := newUser.FindByLogin(callee, s.db)

	if err != nil {
		if errors.As(err, &userNotFound) {
			log.Printf("[INVITE] callee=%s not registered", callee)
			res := sip.NewResponseFromRequest(req, sip.StatusNotFound, "Not Found", nil)
			_ = tx.Respond(res)
			return
		} else {
			res := sip.NewResponseFromRequest(req, sip.StatusInternalServerError, "InternalError", nil)
			_ = tx.Respond(res)
			return
		}
	}

	binding, ok := s.reg.Get(callee)
	if !ok {
		log.Printf("[INVITE] callee=%s not registered", callee)
		res := sip.NewResponseFromRequest(req, sip.StatusNotFound, "Not Found", nil)
		_ = tx.Respond(res)
		return
	}

	log.Printf("[INVITE] route to callee=%s contact=%s (source=%s)", callee, binding.Contact.String(), binding.Source)

	target := sip.Uri{
		Scheme: "sip",
		User:   callee,
		Host:   binding.Contact.Host,
		Port:   binding.Contact.Port,
	}
	target.UriParams = sip.NewParams().Add("transport", "udp")

	if true {
		outBoundInvite := buildOutboundInvite(req, &target, s.host, s.port)

		clTx, err := s.cl.TransactionRequest(context.Background(), outBoundInvite)

		if err != nil {
			res := sip.NewResponseFromRequest(req, sip.StatusServiceUnavailable, "User Unavailable", nil)
			_ = tx.Respond(res)
		}

		go s.proxyInviteResponses(newCtx, clTx)
	} else {
		// 302 + Contact: <sip:callee@ip:port>
		res := sip.NewResponseFromRequest(req, sip.StatusMovedTemporarily, "Moved Temporarily", nil)

		res.AppendHeader(&sip.ContactHeader{
			Address: target,
		})
		tx.Respond(res)
	}

}

func (s *Server) onBye(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	fromTag, _ := req.From().Params.Get("tag")
	toTag, _ := req.To().Params.Get("tag")

	key1, key2 := MakeDialogKey(callID, fromTag, toTag)

	log.Printf("[BYE] Dialog Key: %s  %s", key1, key2)
	v, ok := s.dialogs.Load(key1)
	if !ok {
		v, ok = s.dialogs.Load(key2)
		if !ok {
			log.Printf("[BYE] Dialog key not found")
			_ = tx.Respond(sip.NewResponseFromRequest(req, 481, "Call/Transaction Does Not Exist", nil))
		}
	}
	dlg := v.(*DialogCtx)

	bye := sip.NewRequest(sip.BYE, dlg.RemoteTarget)

	copyFrom := *req.From()
	copyTo := *req.To()
	copyCallId := *req.CallID()
	copyCSeq := *req.CSeq()

	bye.AppendHeader(&copyFrom)
	bye.AppendHeader(&copyTo)
	bye.AppendHeader(&copyCallId)

	if cseq := req.CSeq(); cseq != nil {
		bye.AppendHeader(&copyCSeq)
	}

	var mf sip.MaxForwardsHeader = 70
	bye.AppendHeader(&mf)

	via := &sip.ViaHeader{Transport: "UDP", Host: s.host, Port: s.port, Params: sip.NewParams()}
	via.Params.Add("branch", sip.GenerateBranch())
	via.Params.Add("rport", "")
	via.ProtocolName = "SIP"
	via.ProtocolVersion = "2.0"
	bye.PrependHeader(via)

	clTx, err := s.cl.TransactionRequest(context.Background(), bye)
	if err != nil {
		_ = tx.Respond(sip.NewResponseFromRequest(req, 502, "Bad Gateway", nil))
		return
	}

	select {
	case resp := <-clTx.Responses():
		_ = tx.Respond(sip.NewResponseFromRequest(req, resp.StatusCode, resp.Reason, nil))
		// можно удалить диалог
		s.dialogs.Delete(key1)
		s.dialogs.Delete(key2)
	case <-time.After(3 * time.Second):
		_ = tx.Respond(sip.NewResponseFromRequest(req, 504, "Server Time-out", nil))
	}
}

func makeReachableContact(login string, src string) (sip.Uri, bool) {
	host, portStr, err := net.SplitHostPort(src)
	if err != nil {
		return sip.Uri{}, false
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return sip.Uri{}, false
	}

	u := sip.Uri{
		Scheme: "sip",
		User:   login,
		Host:   host,
		Port:   port,
	}

	u.UriParams = sip.NewParams().Add("transport", "udp")
	return u, true
}

func (s *Server) proxyInviteResponses(ctx *InviteCtx, clTx sip.ClientTransaction) {
	for resp := range clTx.Responses() {
		up := makeUpstreamResponse(ctx.OriginInvite, resp)

		ctx.LastResp = up
		ctx.ServerTx.Respond(up)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 && resp.CSeq() != nil && resp.CSeq().MethodName == sip.INVITE {
			callID := resp.CallID().Value()
			fromTag, _ := resp.From().Params.Get("tag")
			toTag, _ := resp.To().Params.Get("tag")
			callerCT := ctx.OriginInvite.Contact()

			log.Printf("[DIALOG STORE] callid=%s fromTag=%s toTag=%s remote=%s",
				callID, fromTag, toTag, resp.Contact().Address,
			)

			ct := resp.Contact()
			if ct == nil {
				continue
			}

			routes := buildRouteSet(resp)

			if ctx.DialogCreated.CompareAndSwap(false, true) {
				keyAB, keyBA := MakeDialogKey(callID, fromTag, toTag)

				// A -> B (caller -> callee)
				dlgAB := &DialogCtx{
					Key:          keyAB,
					RouteSet:     routes,
					RemoteTarget: ct.Address, // callee contact
				}

				// B -> A (callee -> caller)
				dlgBA := &DialogCtx{
					Key:          keyBA,
					RouteSet:     routes,
					RemoteTarget: callerCT.Address, // caller contact (из INVITE)
				}

				s.dialogs.Store(keyAB, dlgAB)
				s.dialogs.Store(keyBA, dlgBA)
				log.Printf("SAVE DIALOG KEYS: %s %s", keyAB, keyBA)
			}
		}
	}
}
