package router

import (
	"net/http"
	"os"
	"path/filepath"

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

	// web
	dist := "./web/dist"
	fs := http.FileServer(http.Dir(dist))
	r.PathPrefix("/assets/").Handler(fs)
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		p := filepath.Join(dist, req.URL.Path)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			fs.ServeHTTP(w, req)
			return
		}
		http.ServeFile(w, req, filepath.Join(dist, "index.html"))
	})

	return r
}
