package api

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"net/url"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"

	"github.com/undeadops/terse/internal/store"
)

var (
	// keyPattern validates that key is exactly 16 alphanumeric characters
	keyPattern = regexp.MustCompile(`^[a-zA-Z0-9]{16}$`)
	// alphanumeric characters for key generation
	alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type UserHandler struct {
	store  store.Store
	logger zerolog.Logger
}

func Router(ctx context.Context, store store.Store, logger zerolog.Logger) *chi.Mux {
	usr := &UserHandler{
		store:  store,
		logger: logger,
	}

	r := chi.NewRouter()

	r.Use(middleware.Heartbeat("/ping"))
	r.Use(httplog.RequestLogger(logger))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	// Define your routes here, e.g.:
	// r.HandleFunc("/get", s.handleGet)
	// r.Get("/list", s.handleList)
	r.Get("/g/{key}", usr.Redirect)
	r.Route("/manage", func(r chi.Router) {
		r.Post("/", usr.CreateRedirect)
		r.Get("/", usr.ListRedirects)
		r.Delete("/{key}", usr.DeleteRedirect)
	})
	return r
}

func (h *UserHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	// Validate key: must be exactly 16 alphanumeric characters
	if !keyPattern.MatchString(key) {
		http.Error(w, "Invalid key format", http.StatusBadRequest)
		return
	}

	// Call the store to get the value
	value, err := h.store.Get(r.Context(), key)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	url := value.URL
	if url == "" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

type CreateShortURLRequest struct {
	URL       string `json:"url"`
	ExpiresIn int    `json:"expires_in,omitempty"` // in seconds
}

type CreateShortURLResponse struct {
	ShortURL string `json:"short_url,omitempty"`
}

func (c *CreateShortURLRequest) Bind(r *http.Request) error {
	// Validate URL is not empty
	if c.URL == "" {
		return errors.New("url is required")
	}

	// Validate URL format
	parsedURL, err := url.ParseRequestURI(c.URL)
	if err != nil {
		return errors.New("invalid url format")
	}

	// Ensure scheme is http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("url must use http or https scheme")
	}

	// Ensure host is present
	if parsedURL.Host == "" {
		return errors.New("url must have a valid host")
	}

	return nil
}

// generateKey creates a random 16 character alphanumeric string
func generateKey() (string, error) {
	key := make([]byte, 16)
	for i := range key {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphanumeric))))
		if err != nil {
			return "", err
		}
		key[i] = alphanumeric[num.Int64()]
	}
	return string(key), nil
}

func (h *UserHandler) CreateRedirect(w http.ResponseWriter, r *http.Request) {
	// Implementation for creating a new redirect
	data := &CreateShortURLRequest{}
	if err := render.Bind(r, data); err != nil {
		h.handleError(w, err)
		return
	}

	// Generate a unique key and store the URL
	key, _ := generateKey()

	err := h.store.Put(r.Context(), key, data.URL)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shortURL := r.Host + "/g/" + key
	response := &CreateShortURLResponse{ShortURL: shortURL}
	h.respondJSON(w, r, http.StatusCreated, response)
}

type RedirectURLListResponse struct {
	URLs []RedirectURLItem `json:"urls"`
}

type RedirectURLItem struct {
	Key           string `json:"key"`
	URL           string `json:"url"`
	RedirectCount int    `json:"redirect_count"`
}

func (h *UserHandler) ListRedirects(w http.ResponseWriter, r *http.Request) {
	// Implementation for listing all redirects
	data, err := h.store.List(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}

	response := &RedirectURLListResponse{}
	for _, item := range data {
		// Process each item as needed
		response.URLs = append(response.URLs, RedirectURLItem{
			Key:           item.Key,
			URL:           item.URL,
			RedirectCount: item.RedirectCount,
		})
	}

	h.respondJSON(w, r, http.StatusOK, response)
}

type DeleteRedirectResponse struct {
	Message string `json:"message"`
}

type DeleteRedirectRequest struct {
	Key string `json:"key"`
}

func (h *UserHandler) DeleteRedirect(w http.ResponseWriter, r *http.Request) {
	// Implementation for deleting a redirect
	key := chi.URLParam(r, "key")

	err := h.store.Delete(r.Context(), key)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response := &DeleteRedirectResponse{Message: "Redirect deleted successfully"}
	h.respondJSON(w, r, http.StatusOK, response)
}

// Helper methods for consistent error handling and responses
func (h *UserHandler) handleError(w http.ResponseWriter, err error) {
	// Convert different error types to appropriate HTTP responses
	h.logger.Error().Err(err).Msg("Handling error")
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func (h *UserHandler) respondJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	// Common response formatting
	w.WriteHeader(status)
	render.JSON(w, r, data)
}
