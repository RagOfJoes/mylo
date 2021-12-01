package transport

import (
	"context"
	"net/http"

	"github.com/RagOfJoes/mylo/internal"
	"github.com/RagOfJoes/mylo/internal/config"
	"github.com/RagOfJoes/mylo/internal/validate"
	"github.com/RagOfJoes/mylo/session"
	"github.com/gorilla/sessions"
)

type Http struct {
	cfg config.Configuration

	st sessions.Store
	se session.Service
}

func NewSessionHttp(cfg config.Configuration, st sessions.Store, se session.Service) *Http {
	return &Http{
		cfg: cfg,

		st: st,
		se: se,
	}
}

// New creates a new Unauthenticated session and stores it in Repository
func (h *Http) New(ctx context.Context, req *http.Request, w http.ResponseWriter) (*session.Session, error) {
	newSession, err := session.NewUnauthenticated()
	if err != nil {
		return nil, err
	}
	created, err := h.se.New(ctx, *newSession)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// Upsert will update a session if it exists and if not then it will create a new one
func (h *Http) Upsert(ctx context.Context, upsertSession session.Session) (*session.Session, error) {
	_, err := h.se.FindByID(ctx, upsertSession.ID)
	if err != nil {
		newSess, err := h.se.New(ctx, upsertSession)
		if err != nil {
			return nil, err
		}
		return newSess, nil
	}
	updated, err := h.se.Update(ctx, upsertSession)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// SetCookie sets provided session into cookie
func (h *Http) SetCookie(req *http.Request, w http.ResponseWriter, s session.Session) error {
	cookie, err := h.st.Get(req, h.cfg.Session.Cookie.Name)
	if err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to retrieve session from cookie store")
	}
	if err := validate.Check(s); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", session.ErrInvalidSession)
	}
	cookie.Values["session"] = s.Token
	if err := cookie.Save(req, w); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to set session token into cookie")
	}
	return nil
}

// UpsertAndSetCookie will call Upsert method then SetCookie method
func (h *Http) UpsertAndSetCookie(ctx context.Context, req *http.Request, w http.ResponseWriter, upsertSession session.Session) (*session.Session, error) {
	upserted, err := h.Upsert(ctx, upsertSession)
	if err != nil {
		return nil, err
	}
	if err := h.SetCookie(req, w, upsertSession); err != nil {
		return nil, err
	}
	return upserted, nil
}

// Session retrieves session token from Request then fetches from service.
//
// If mustBeAuthenticated is true then will return error if session is not authenticated
func (h *Http) Session(ctx context.Context, req *http.Request, w http.ResponseWriter, mustBeAuthenticated bool) (*session.Session, error) {
	token := h.getToken(req)
	if token == "" {
		return nil, internal.NewErrorf(internal.ErrorCodeUnauthorized, "%v", session.ErrSessionNotFound)
	}
	found, err := h.se.FindByToken(ctx, token)
	if err != nil || (mustBeAuthenticated && found != nil && !found.Authenticated()) {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnauthorized, "%v", session.ErrSessionNotFound)
	}
	if found.IdentityID != nil && found.Identity != nil {
		found.Identity.Credentials = nil
	}
	return found, nil
}

// SessionOrNew will retrieve a session if it exists and if not then create a new one. If a new session needs to be created then mustBeAuthenticated is ignored
func (h *Http) SessionOrNew(ctx context.Context, req *http.Request, w http.ResponseWriter, mustBeAuthenticated bool) (*session.Session, error) {
	token := h.getToken(req)
	if token == "" {
		return nil, internal.NewErrorf(internal.ErrorCodeUnauthorized, "%v", session.ErrSessionNotFound)
	}
	found, _ := h.Session(ctx, req, w, false)
	if found == nil {
		newSession, err := h.New(ctx, req, w)
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
func (h *Http) SessionOrNewAndSetCookie(ctx context.Context, req *http.Request, w http.ResponseWriter, mustBeAuthenticated bool) (*session.Session, error) {
	found, _ := h.Session(ctx, req, w, false)
	if found == nil {
		newSession, err := h.New(ctx, req, w)
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
	cookie, err := h.st.Get(req, h.cfg.Session.Cookie.Name)
	if err != nil {
		return ""
	}
	token, ok := cookie.Values["session"].(string)
	if !ok {
		return ""
	}
	return token
}
