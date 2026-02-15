package registrar

import (
	"reflect"
	"sync"
	"time"

	"github.com/emiago/sipgo/sip"
)

type ContactBinding struct {
	Contact   sip.Uri
	ExpiresAt time.Time
	Source    string // host:port откуда пришёл запрос (для отладки)
}

type Registrar struct {
	mu  sync.RWMutex
	loc map[string]ContactBinding // user -> binding
	ttl time.Duration
}

func New(ttl time.Duration) *Registrar {
	return &Registrar{
		loc: make(map[string]ContactBinding),
		ttl: ttl,
	}
}

func (r *Registrar) Put(user string, contact sip.Uri, source string, expires time.Duration) {
	if expires <= 0 {
		expires = r.ttl
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loc[user] = ContactBinding{
		Contact:   contact,
		ExpiresAt: time.Now().Add(expires),
		Source:    source,
	}

}

func (r *Registrar) IsRegistered(user string, contact sip.Uri, source string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if contactBinding, ok := r.loc[user]; ok {
		if reflect.DeepEqual(contactBinding, contact) && contactBinding.Source == source && !time.Now().After(contactBinding.ExpiresAt) {
			return true
		}
	}

	return false
}

func (r *Registrar) Get(user string) (ContactBinding, bool) {
	r.mu.RLock()
	b, ok := r.loc[user]
	r.mu.RUnlock()
	if !ok {
		return ContactBinding{}, false
	}
	if time.Now().After(b.ExpiresAt) {
		r.mu.Lock()
		delete(r.loc, user)
		r.mu.Unlock()
		return ContactBinding{}, false
	}
	return b, true
}

func (r *Registrar) Delete(user string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.loc, user)
}
