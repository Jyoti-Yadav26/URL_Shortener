package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Jyoti-Yadav26/URL_Shortener/internal/service"
)

// maxBodyBytes caps the request body so a malicious client cannot make the
// server read without limit.
const maxBodyBytes = 8 << 10

type shortenRequest struct {
	URL            string `json:"url"`
	Alias          string `json:"alias"`
	ExpiresInHours int    `json:"expires_in_hours"`
}

type shortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

type URLHandler struct {
	svc     *service.Service
	baseURL string
}

func NewURLHandler(svc *service.Service, baseURL string) *URLHandler {
	return &URLHandler{svc: svc, baseURL: strings.TrimSuffix(baseURL, "/")}
}

func (h *URLHandler) Shorten(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be a JSON object")
		return
	}

	stored, err := h.svc.Shorten(r.Context(), req.URL, req.Alias, req.ExpiresInHours)
	if err != nil {
		writeShortenError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, shortenResponse{
		Code:     stored.Code,
		ShortURL: h.baseURL + "/" + stored.Code,
	})
}

func (h *URLHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stored, err := h.svc.Resolve(r.Context(), code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound, "not_found", "no such short link")
		case errors.Is(err, service.ErrExpired):
			writeError(w, http.StatusGone, "expired", "this short link has expired")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		}
		return
	}

	// Without this a cache could answer later requests itself, and an
	// expired link would keep redirecting.
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, stored.TargetURL, http.StatusFound)
}

func writeShortenError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidURL):
		writeError(w, http.StatusBadRequest, "invalid_url", "url must be an http or https address")
	case errors.Is(err, service.ErrInvalidAlias):
		writeError(w, http.StatusBadRequest, "invalid_alias",
			"alias must be 3 to 32 characters of letters, digits, hyphen or underscore")
	case errors.Is(err, service.ErrInvalidExpiry):
		writeError(w, http.StatusBadRequest, "invalid_expiry", "expires_in_hours must not be negative")
	case errors.Is(err, service.ErrAliasTaken):
		writeError(w, http.StatusConflict, "alias_taken", "that alias is already in use")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}
