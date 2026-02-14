package router

import (
	"github.com/gorilla/mux"

	httpserver "SipServer/internal/http_server"
)

func NewRouter(s *httpserver.HttpServer) *mux.Router {
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	// users
	api.HandleFunc("/users", s.ListUsers).Methods("GET")
	api.HandleFunc("/users/{id:[0-9]+}", s.GetUser).Methods("GET")
	api.HandleFunc("/users", s.CreateUser).Methods("POST")
	api.HandleFunc("/users/{id:[0-9]+}", s.UpdateUser).Methods("PUT")
	// sessions
	api.HandleFunc("/sessions", s.ListSession).Methods("GET")
	// call_journals
	api.HandleFunc("/call_journals", s.ListCallJournal).Methods("GET")

	return r
}
