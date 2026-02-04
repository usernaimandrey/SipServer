package sipserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"

	"SipServer/internal/entity/user"
	"SipServer/internal/registrar"
)

var userNotFound *user.UserNotFoundError

type Server struct {
	srv      *sipgo.Server
	cli      *sipgo.Client
	reg      *registrar.Registrar
	db       *sql.DB
	UserRepo *user.UserRepo

	publicHost string
	publicPort int

	mu       sync.Mutex
	invTxMap map[inviteKey]*inviteCtx
}

type inviteKey struct {
	callID string
	cseqNo string
}

type inviteCtx struct {
	inviteReq *sip.Request
	dest      string
}

func New(
	ua *sipgo.UserAgent,
	reg *registrar.Registrar,
	db *sql.DB,
	publicHost string,
	publicPort int,
	userRepo *user.UserRepo,
) (*Server, error) {
	srv, err := sipgo.NewServer(ua)
	if err != nil {
		return nil, err
	}

	cli, err := sipgo.NewClient(
		ua,
		sipgo.WithClientHostname(publicHost),
		sipgo.WithClientPort(publicPort),
	)

	if err != nil {
		return nil, err
	}

	s := &Server{
		srv:        srv,
		reg:        reg,
		db:         db,
		cli:        cli,
		publicHost: publicHost,
		publicPort: publicPort,
		UserRepo:   userRepo,
		invTxMap:   make(map[inviteKey]*inviteCtx),
	}

	// REGISTER / INVITE / BYE — ключевые методы для прототипа
	srv.OnRegister(s.onRegister) // хендлеры вида func(req *sip.Request, tx sip.ServerTransaction) :contentReference[oaicite:1]{index=1}
	srv.OnInvite(s.onInvite)
	srv.OnBye(s.onInDialogProxy)
	srv.OnAck(s.onInDialogProxy)
	srv.OnCancel(s.onCancel)

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
	// В перспективе: 401 + WWW-Authenticate и проверка Authorization.

	from := req.From()
	contact := req.Contact()

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

	_, err := s.UserRepo.FindByLogin(login)

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
		s.reg.Put(login, contact.Address, src, 60*time.Second)
		log.Printf("[REGISTER] user=%s contact=%s (raw) source=%s",
			login, contact.Address.String(), src)
	}

	s.reg.Put(login, contact.Address, req.Source(), 60*time.Second)

	log.Printf("[REGISTER] user=%s contact=%s source=%s", login, contact.Address.String(), req.Source())

	res := sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil)
	_ = tx.Respond(res)
}

func (s *Server) onInvite(req *sip.Request, tx sip.ServerTransaction) {
	_ = tx.Respond(sip.NewResponseFromRequest(req, sip.StatusTrying, "Trying", nil)) // пример Respond :contentReference[oaicite:3]{index=3}

	to := req.To()

	if to == nil || strings.TrimSpace(to.Address.User) == "" {
		res := sip.NewResponseFromRequest(req, sip.StatusBadRequest, "Bad Request", nil)
		_ = tx.Respond(res)
		return
	}
	callee := strings.TrimSpace(to.Address.User)

	_, err := s.UserRepo.FindByLogin(callee)

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

	target := sip.Uri{
		Scheme: "sip",
		User:   callee,
		Host:   binding.Contact.Host,
		Port:   binding.Contact.Port,
	}
	target.UriParams = sip.NewParams().Add("transport", "udp")

	log.Printf("[INVITE] route to callee=%s contact=%s (source=%s)", callee, binding.Contact.String(), binding.Source)

	out := req.Clone()

	setRequestURIAndDest(out, target)
	decreaseMaxForwards(out)
	s.addTopVia(out)
	s.addRecordRoute(out)

	key := makeInviteKey(req)

	s.mu.Lock()
	s.invTxMap[key] = &inviteCtx{
		inviteReq: out.Clone(),
		dest:      out.Destination(),
	}
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)

	defer cancel()

	ctxTx, err := s.cli.TransactionRequest(ctx, out)

	if err != nil {
		_ = tx.Respond(sip.NewResponseFromRequest(req, sip.StatusServiceUnavailable, "Service Unavailable", nil))
		return
	}

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.invTxMap, key)
			s.mu.Unlock()
		}()

		for resp := range ctxTx.Responses() {
			r := resp.Clone()

			stripTopVia(r) // <-- вместо полного RemoveHeader/Prepend чужого Via

			_ = tx.Respond(r)

			if r.StatusCode >= 200 {
				return
			}
		}

	}()

}

func (s *Server) onInDialogProxy(req *sip.Request, stx sip.ServerTransaction) {
	out := req.Clone()

	// Если первый Route = мы, снимаем (чтобы не зациклиться)
	if r := out.Route(); r != nil {
		if strings.EqualFold(r.Address.Host, s.publicHost) && r.Address.Port == s.publicPort {
			out.RemoveHeader("Route")
		}
	}

	// Выбираем next hop: первый Route, иначе Request-URI (Recipient)
	if r := out.Route(); r != nil {
		out.SetDestination(r.Address.HostPort())
	} else {
		out.SetDestination(out.Recipient.HostPort())
	}

	decreaseMaxForwards(out)
	s.addTopVia(out)

	if out.Method == sip.ACK {
		// ACK не транзакционный
		if err := s.cli.WriteRequest(out); err != nil {
			log.Printf("[ACK] forward failed err=%v", err)
		}
		log.Printf("[ACK] forward succcess out=%v", out)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ctxTx, err := s.cli.TransactionRequest(ctx, out)
	log.Println("[Response %s]", ctxTx)
	if err != nil {
		_ = stx.Respond(sip.NewResponseFromRequest(req, sip.StatusServiceUnavailable, "Service Unavailable", nil))
		return
	}

	go func() {
		for resp := range ctxTx.Responses() {
			r := resp.Clone()
			stripTopVia(r) // <-- и тут тоже, чтобы ответы шли назад правильно
			_ = stx.Respond(r)
		}
	}()
}

func (s *Server) onBye(req *sip.Request, tx sip.ServerTransaction) {

	log.Printf("[BYE] call-id=%v source=%s", safeCallID(req), req.Source())

	res := sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil)
	_ = tx.Respond(res)
}

func safeCallID(req *sip.Request) string {
	if req == nil || req.CallID() == nil {
		return ""
	}
	return fmt.Sprintf("%v", req.CallID())
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

func (s *Server) addTopVia(req *sip.Request) {
	via := &sip.ViaHeader{
		ProtocolName:    "SIP",
		ProtocolVersion: "2.0",
		Transport:       "UDP",
		Host:            s.publicHost,
		Port:            s.publicPort,
		Params: sip.NewParams().
			Add("branch", sip.GenerateBranch()).
			Add("rport", ""),
	}
	req.PrependHeader(via)
}

func (s *Server) addRecordRoute(req *sip.Request) {
	rrURI := sip.Uri{
		Scheme: "sip",
		Host:   s.publicHost,
		Port:   s.publicPort,
		UriParams: sip.NewParams().
			Add("lr", ""), // loose routing
	}

	req.AppendHeader(&sip.RecordRouteHeader{Address: rrURI})
}

func (s *Server) onCancel(req *sip.Request, stx sip.ServerTransaction) {
	// RFC 3261: на CANCEL всегда отвечаем 200 OK
	_ = stx.Respond(sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil))

	key := makeInviteKey(req)

	s.mu.Lock()
	ictx := s.invTxMap[key]
	s.mu.Unlock()

	if ictx == nil {
		log.Printf("[CANCEL] no matching INVITE callid=%s cseq=%s", key.callID, key.cseqNo)
		return
	}

	// --- строим CANCEL ---
	cancelReq := sip.NewRequest(sip.CANCEL, ictx.inviteReq.Recipient)

	// Via (тот же branch!)
	origVia := ictx.inviteReq.Via()
	cancelReq.AppendHeader(&sip.ViaHeader{
		ProtocolName:    origVia.ProtocolName,
		ProtocolVersion: origVia.ProtocolVersion,
		Transport:       origVia.Transport,
		Host:            origVia.Host,
		Port:            origVia.Port,
		Params:          origVia.Params,
	})

	// From
	origFrom := ictx.inviteReq.From()
	cancelReq.AppendHeader(&sip.FromHeader{
		Address: origFrom.Address,
		Params:  origFrom.Params,
	})

	// To
	origTo := ictx.inviteReq.To()
	cancelReq.AppendHeader(&sip.ToHeader{
		Address: origTo.Address,
		Params:  origTo.Params,
	})

	origCallID := ictx.inviteReq.CallID()
	cid := sip.CallIDHeader(origCallID.Value())
	cancelReq.AppendHeader(&cid)

	// CSeq: тот же номер, но метод CANCEL
	origCSeq := ictx.inviteReq.CSeq()
	cancelReq.AppendHeader(&sip.CSeqHeader{
		SeqNo:      origCSeq.SeqNo,
		MethodName: sip.CANCEL,
	})

	// куда реально слать
	cancelReq.SetDestination(ictx.dest)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := s.cli.TransactionRequest(ctx, cancelReq); err != nil {
		log.Printf("[CANCEL] downstream cancel failed callid=%s err=%v", key.callID, err)
		return
	}

	log.Printf("[CANCEL] forwarded downstream callid=%s cseq=%s", key.callID, key.cseqNo)
}
