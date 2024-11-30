package caddypocketbase

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(new(adminAPI))
}

// adminAPI is a module that serves PKI endpoints to retrieve
// information about the CAs being managed by Caddy.
type adminAPI struct {
	ctx caddy.Context
	log *zap.Logger
	app *App
}

// CaddyModule returns the Caddy module information.
func (adminAPI) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "admin.api.pocketbase",
		New: func() caddy.Module { return new(adminAPI) },
	}
}

// Provision sets up the adminAPI module.
func (a *adminAPI) Provision(ctx caddy.Context) error {
	a.ctx = ctx
	a.log = ctx.Logger(a) // TODO: passing in 'a' is a hack until the admin API is officially extensible (see #5032)

	pbApp, err := ctx.App("pocketbase")
	if err != nil {
		return err
	}

	a.app = pbApp.(*App)

	return nil
}

// Routes returns the admin routes for the PKI app.
func (a *adminAPI) Routes() []caddy.AdminRoute {
	return []caddy.AdminRoute{
		{
			Pattern: adminEndpointBase,
			Handler: caddy.AdminHandlerFunc(a.handleAPIEndpoints),
		},
	}
}

// handleAPIEndpoints routes API requests within handleAPIEndpoints.
func (a *adminAPI) handleAPIEndpoints(w http.ResponseWriter, r *http.Request) error {
	uri := strings.TrimPrefix(r.URL.Path, "/pocketbase/")
	parts := strings.Split(uri, "/")
	switch {
	case len(parts) == 1 && strings.EqualFold(parts[0], "superuser"):
		return a.handleSuperuser(w, r)
	case len(parts) == 3 && strings.EqualFold(parts[0], "superuser") && strings.EqualFold(parts[2], "otp"):
		return a.handleSuperuserOTP(parts[1], w, r)
	}
	return caddy.APIError{
		HTTPStatus: http.StatusNotFound,
		Err:        fmt.Errorf("resource not found: %v", r.URL.Path),
	}
}

func (a *adminAPI) handleSuperuser(w http.ResponseWriter, r *http.Request) error {
	var sr superuserRequest
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	if err := decoder.Decode(&sr); err != nil {
		return caddy.APIError{
			HTTPStatus: http.StatusBadRequest,
			Err:        fmt.Errorf("payload unintelligible: %v", err),
		}
	}

	if is.EmailFormat.Validate(sr.EmailAddress) != nil {
		return caddy.APIError{
			HTTPStatus: http.StatusBadRequest,
			Err:        errors.New("invalid or missing email address"),
		}
	}

	switch r.Method {
	case http.MethodPost:
		superusersCol, err := a.app.pb.FindCachedCollectionByNameOrId(core.CollectionNameSuperusers)
		if err != nil {
			return fmt.Errorf("Failed to fetch %q collection: %w.", core.CollectionNameSuperusers, err)
		}

		superuser := core.NewRecord(superusersCol)
		superuser.SetEmail(sr.EmailAddress)
		superuser.SetPassword(sr.Password)

		if err := a.app.pb.Save(superuser); err != nil {
			return fmt.Errorf("Failed to create new superuser account: %w.", err)
		}
		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		superuser, err := a.app.pb.FindAuthRecordByEmail(core.CollectionNameSuperusers, sr.EmailAddress)
		if err != nil {
			return nil
		}

		if err := a.app.pb.Delete(superuser); err != nil {
			return fmt.Errorf("Failed to delete superuser %q: %w.", superuser.Email(), err)
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodPut: // upsert
		superusersCol, err := a.app.pb.FindCachedCollectionByNameOrId(core.CollectionNameSuperusers)
		if err != nil {
			return fmt.Errorf("Failed to fetch %q collection: %w.", core.CollectionNameSuperusers, err)
		}

		superuser, err := a.app.pb.FindAuthRecordByEmail(superusersCol, sr.EmailAddress)
		if err != nil {
			superuser = core.NewRecord(superusersCol)
		}

		superuser.SetEmail(sr.EmailAddress)
		superuser.SetPassword(sr.Password)

		if err := a.app.pb.Save(superuser); err != nil {
			return fmt.Errorf("Failed to upsert superuser account: %w.", err)
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodPatch: // update
		superuser, err := a.app.pb.FindAuthRecordByEmail(core.CollectionNameSuperusers, sr.EmailAddress)
		if err != nil {
			return fmt.Errorf("Superuser with email %q doesn't exist.", sr.EmailAddress)
		}

		superuser.SetPassword(sr.Password)

		if err := a.app.pb.Save(superuser); err != nil {
			return fmt.Errorf("Failed to change superuser %q password: %w.", superuser.Email(), err)
		}
		w.WriteHeader(http.StatusOK)
	default:
		return caddy.APIError{
			HTTPStatus: http.StatusMethodNotAllowed,
		}
	}
	return nil
}

func (a *adminAPI) handleSuperuserOTP(middle string, w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return caddy.APIError{
			HTTPStatus: http.StatusMethodNotAllowed,
		}
	}
	if len(middle) == 0 || is.EmailFormat.Validate(middle) != nil {
		return caddy.APIError{
			HTTPStatus: http.StatusBadRequest,
			Err:        fmt.Errorf("invalid or missing email address"),
		}
	}

	superuser, err := a.app.pb.FindAuthRecordByEmail(core.CollectionNameSuperusers, middle)
	if err != nil {
		return caddy.APIError{
			HTTPStatus: http.StatusNotFound,
			Err:        fmt.Errorf("Superuser with email %q doesn't exist.", middle),
		}
	}

	if !superuser.Collection().OTP.Enabled {
		return errors.New("OTP is not enabled for the _superusers collection.")
	}

	pass := security.RandomStringWithAlphabet(superuser.Collection().OTP.Length, "1234567890")

	otp := core.NewOTP(a.app.pb)
	otp.SetCollectionRef(superuser.Collection().Id)
	otp.SetRecordRef(superuser.Id)
	otp.SetPassword(pass)

	err = a.app.pb.Save(otp)
	if err != nil {
		return fmt.Errorf("Failed to create OTP: %w", err)
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

type superuserRequest struct {
	EmailAddress string `json:"email_address,omitempty"`
	Password     string `json:"password,omitempty"`
}

const adminEndpointBase = "/pocketbase/"
