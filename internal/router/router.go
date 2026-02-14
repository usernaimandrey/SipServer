package router

import (
	"github.com/gorilla/mux"

	httpserver "SipServer/internal/http_server"
)

func NewRouter(s *httpserver.HttpServer) *mux.Router {
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/users", s.ListUsers).Methods("GET")
	api.HandleFunc("/users/{id:[0-9]+}", s.GetUser).Methods("GET")
	api.HandleFunc("/users", s.CreateUser).Methods("POST")
	api.HandleFunc("/users/{id:[0-9]+}", s.UpdateUser).Methods("PUT")

	return r
}
