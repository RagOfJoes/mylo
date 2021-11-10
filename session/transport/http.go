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

// Upsert will update a session if it exists and if not then it will create a new one
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

// UpsertAndSetCookie will call Upsert method then SetCookie method
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

// Session retrieves session token from Request then fetches from service.
//
// If mustBeAuthenticated is true then will return error if session is not authenticated
func (h *Http) Session(req *http.Request, w http.ResponseWriter, mustBeAuthenticated bool) (*session.Session, error) {
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

// SessionOrNew will retrieve a session if it exists and if not then create a new one. If a new session needs to be created then mustBeAuthenticated is ignored
func (h *Http) SessionOrNew(req *http.Request, w http.ResponseWriter, mustBeAuthenticated bool) (*session.Session, error) {
	token := h.getToken(req)
	if token == "" {
		return nil, internal.NewErrorf(internal.ErrorCodeUnauthorized, "%v", session.ErrSessionNotFound)
	}
	found, _ := h.Session(req, w, false)
	if found == nil {
		newSession, err := h.New(req, w)
		if err != nil {
			return nil, err
		}
		return newSession, nil
	}
	if mustBeAuthenticated && found != nil && !found.Authenticated() {
		return nil, internal.NewErrorf(internal.ErrorCodeUnauthorized, "%v", session.ErrSessionNotFound)
	}
	return found, nil
}

// SessionOrNewAndSetCookie does exactly what SessionOrNew does except it sets a session cookie when a new session needs to be created
func (h *Http) SessionOrNewAndSetCookie(req *http.Request, w http.ResponseWriter, mustBeAuthenticated bool) (*session.Session, error) {
	found, _ := h.Session(req, w, false)
	if found == nil {
		newSession, err := h.New(req, w)
		if err != nil {
			return nil, err
		}
		if err := h.SetCookie(req, w, *newSession); err != nil {
			return nil, err
		}
		return newSession, nil
	}
	if mustBeAuthenticated && found != nil && !found.Authenticated() {
		return nil, internal.NewErrorf(internal.ErrorCodeUnauthorized, "%v", session.ErrSessionNotFound)
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
