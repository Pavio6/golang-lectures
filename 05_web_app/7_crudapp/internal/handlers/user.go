package handlers

import (
	"errors"
	"html/template"
	"net/http"

	"crudapp/internal/session"
	"crudapp/internal/user"

	"go.uber.org/zap"
)

type UserHandler struct {
	Tmpl     *template.Template
	Logger   *zap.SugaredLogger
	UserRepo user.UserRepo
	Sessions *session.SessionsManager
}

func (h *UserHandler) Index(w http.ResponseWriter, r *http.Request) {
	_, err := session.SessionFromContext(r.Context())
	if err == nil {
		http.Redirect(w, r, "/items", http.StatusSeeOther)
		return
	}

	err = h.Tmpl.ExecuteTemplate(w, "login.html", nil)
	if err != nil {
		http.Error(w, `Template errror`, http.StatusInternalServerError)
		return
	}
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	u, err := h.UserRepo.Authorize(r.FormValue("login"), r.FormValue("password"))
	if errors.Is(err, user.ErrNoUser) {
		http.Error(w, `no user`, http.StatusBadRequest)
		return
	}
	if errors.Is(err, user.ErrBadPass) {
		http.Error(w, `bad pass`, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "authorization failed", http.StatusInternalServerError)
		return
	}

	sess, err := h.Sessions.Create(w, u.ID)
	if err != nil {
		http.Error(w, "could not create session", http.StatusInternalServerError)
		return
	}
	h.Logger.Infof("created session for %v", sess.UserID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.Sessions.DestroyCurrent(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
