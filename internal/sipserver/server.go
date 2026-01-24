package sipserver

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"

	"SipServer/internal/registrar"
)

type Server struct {
	srv *sipgo.Server
	reg *registrar.Registrar
}

func New(ua *sipgo.UserAgent, reg *registrar.Registrar) (*Server, error) {
	srv, err := sipgo.NewServer(ua)
	if err != nil {
		return nil, err
	}

	s := &Server{
		srv: srv,
		reg: reg,
	}

	// REGISTER / INVITE / BYE — ключевые методы для прототипа
	srv.OnRegister(s.onRegister) // хендлеры вида func(req *sip.Request, tx sip.ServerTransaction) :contentReference[oaicite:1]{index=1}
	srv.OnInvite(s.onInvite)
	srv.OnBye(s.onBye)

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

	user := strings.TrimSpace(from.Address.User)
	if user == "" {
		res := sip.NewResponseFromRequest(req, sip.StatusBadRequest, "Bad Request", nil)
		_ = tx.Respond(res)
		return
	}

	// Параметр expires: в SIP бывает и в заголовке Expires, и в Contact; для прототипа возьмём дефолт.
	s.reg.Put(user, contact.Address, req.Source(), 60*time.Second)

	log.Printf("[REGISTER] user=%s contact=%s source=%s", user, contact.Address.String(), req.Source())

	res := sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil)
	_ = tx.Respond(res)
}

func (s *Server) onInvite(req *sip.Request, tx sip.ServerTransaction) {
	// Прототипный сценарий:
	// 1) 100 Trying
	// 2) проверяем, зарегистрирован ли вызываемый абонент
	// 3) если да — возвращаем 302 Moved Temporarily + Contact (редирект на устройство)
	//    (это простой способ показать "маршрутизацию" без проксирования диалога)
	// 4) если нет — 404 Not Found

	_ = tx.Respond(sip.NewResponseFromRequest(req, sip.StatusTrying, "Trying", nil)) // пример Respond :contentReference[oaicite:3]{index=3}

	to := req.To()
	if to == nil || strings.TrimSpace(to.Address.User) == "" {
		res := sip.NewResponseFromRequest(req, sip.StatusBadRequest, "Bad Request", nil)
		_ = tx.Respond(res)
		return
	}
	callee := strings.TrimSpace(to.Address.User)

	binding, ok := s.reg.Get(callee)
	if !ok {
		log.Printf("[INVITE] callee=%s not registered", callee)
		res := sip.NewResponseFromRequest(req, sip.StatusNotFound, "Not Found", nil)
		_ = tx.Respond(res)
		return
	}

	log.Printf("[INVITE] route to callee=%s contact=%s (source=%s)", callee, binding.Contact.String(), binding.Source)

	// 302 + Contact: <sip:callee@ip:port>
	res := sip.NewResponseFromRequest(req, sip.StatusMovedTemporarily, "Moved Temporarily", nil)

	// В ответ добавляем Contact
	res.AppendHeader(&sip.ContactHeader{ // структура ContactHeader: DisplayName/Address/Params :contentReference[oaicite:4]{index=4}
		Address: binding.Contact,
	})

	_ = tx.Respond(res)
}

func (s *Server) onBye(req *sip.Request, tx sip.ServerTransaction) {
	// В редирект-модели BYE обычно пойдёт напрямую между абонентами,
	// но метод поддерживаем, чтобы соответствовать требованиям прототипа.
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
