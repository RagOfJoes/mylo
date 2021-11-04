package transport

import (
	"net/http"

	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/session"
	"github.com/RagOfJoes/idp/transport"
	"github.com/gorilla/sessions"
)

type Http struct {
	st sessions.Store
	se session.Service
}

func NewSessionHttp(st sessions.Store, se session.Service) *Http {
	return &Http{
		st: st,
		se: se,
	}
}

// New creates a new Unauthenticated session and stores it in Repository
func (h *Http) New(req *http.Request, w http.ResponseWriter) (*session.Session, error) {
	newSession, err := session.NewUnauthenticated()
	if err != nil {
		return nil, err
	}
	created, err := h.se.New(*newSession)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (h *Http) Upsert(s session.Session) (*session.Session, error) {
	_, err := h.se.FindByID(s.ID)
	if err != nil {
		newSess, err := h.se.New(s)
		if err != nil {
			return nil, err
		}
		return newSess, nil
	}
	updated, err := h.se.Update(s)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// SetCookie sets provided session into cookie
func (h *Http) SetCookie(req *http.Request, w http.ResponseWriter, s session.Session) error {
	cfg := config.Get()
	cookie, err := h.st.Get(req, cfg.Session.Cookie.Name)
	if err != nil {
		return err
	}
	if err := validate.Check(s); err != nil {
		return err
	}
	cookie.Values["session"] = s.Token
	if err := cookie.Save(req, w); err != nil {
		return err
	}
	return nil
}

func (h *Http) UpsertAndSetCookie(req *http.Request, w http.ResponseWriter, s session.Session) (*session.Session, error) {
	upserted, err := h.Upsert(s)
	if err != nil {
		return nil, err
	}
	if err := h.SetCookie(req, w, s); err != nil {
		return nil, err
	}
	return upserted, nil
}

// Session retrieves session token from Request then fetches from service
func (h *Http) Session(req *http.Request, w http.ResponseWriter) (*session.Session, error) {
	token := h.getToken(req)
	if token == "" {
		return nil, transport.NewHttpClientError(nil, http.StatusUnauthorized, "Session_NotFound", "No active session", nil)
	}

	found, err := h.se.FindByToken(token)
	if err != nil || (found != nil && !found.Authenticated()) {
		return nil, transport.NewHttpClientError(err, http.StatusUnauthorized, "Session_NotFound", "No active session", nil)
	}
	if found.IdentityID != nil && found.Identity != nil {
		found.Identity.Credentials = nil
	}
	return found, nil
}

func (h *Http) getToken(req *http.Request) string {
	// First check Headers
	if token := req.Header.Get("X-Session-Token"); token != "" {
		return token
	}
	// Then check for cookie
	cfg := config.Get()
	cookie, err := h.st.Get(req, cfg.Session.Cookie.Name)
	if err != nil {
		return ""
	}
	token, ok := cookie.Values["session"].(string)
	if !ok {
		return ""
	}
	return token
}
