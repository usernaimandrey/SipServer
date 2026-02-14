package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"SipServer/internal/repository/user"
	"SipServer/internal/usecase"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

type HttpServer struct {
	userUsecase        *usecase.UserUsecase
	sessionUsecase     *usecase.SessionUsecase
	callJournalUsecase *usecase.CallJournalUsecase
	validator          *validator.Validate
}

func NewHttpServer(db *sql.DB) *HttpServer {
	return &HttpServer{
		userUsecase:        usecase.NewUserUseCase(db),
		sessionUsecase:     usecase.NewSessionUsecase(db),
		callJournalUsecase: usecase.NewCallJournalUsecase(db),
		validator:          validator.New(),
	}
}

func (s *HttpServer) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.userUsecase.ListUsers()
	buildResponse(users, w, err)
}

func (s *HttpServer) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	user, err := s.userUsecase.GetUser(id)

	buildResponse(user, w, err)
}

func (s *HttpServer) CreateUser(w http.ResponseWriter, r *http.Request) {
	body := r.Body

	defer body.Close()
	user := user.NewUser()

	err := json.NewDecoder(body).Decode(user)
	if err != nil {
		buildResponse(struct{}{}, w, err)
		return
	}

	err = s.validator.Struct(user)
	if err != nil {
		buildResponse(user, w, err)
		return
	}

	user, err = s.userUsecase.CreateUser(user)
	buildResponse(user, w, err)
}

func (s *HttpServer) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	req := user.NewUserUpdateReq()

	body := r.Body

	err := json.NewDecoder(body).Decode(req)
	if err != nil {
		buildResponse(struct{}{}, w, err)
		return
	}

	err = s.validator.Struct(req)
	if err != nil {
		buildResponse(req, w, err)
		return
	}

	err = s.userUsecase.UpdateUser(id, req)
	buildResponse(req, w, err)
}

func (s *HttpServer) ListSession(w http.ResponseWriter, _ *http.Request) {
	session, err := s.sessionUsecase.List()
	buildResponse(session, w, err)
}

func (s *HttpServer) ListCallJournal(w http.ResponseWriter, _ *http.Request) {
	call_journals, err := s.callJournalUsecase.List()
	buildResponse(call_journals, w, err)
}

func buildResponse(entity interface{}, w http.ResponseWriter, err error) {
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]interface{}{
				"errors": map[string]interface{}{
					"user": "user not found",
				},
			})
			return
		}
		errors, ok := err.(validator.ValidationErrors)
		if ok {
			errorsMap := map[string]interface{}{}
			for _, e := range errors {
				var errText string
				switch e.Tag() {
				case "required":
					errText = "field is required"
				case "oneof":
					errText = fmt.Sprintf("fiels is oneof %s", e.Param())
				case "min":
					errText = fmt.Sprintf("field min len %s", e.Param())
				case "max":
					errText = fmt.Sprintf("field min len %s", e.Param())
				default:
					errText = "invalid value"
				}

				errorsMap[e.Field()] = errText
			}
			writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
				"errors": errorsMap,
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"errors": map[string]interface{}{
				"server": fmt.Sprintf("internal server error %v", err),
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": entity,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
