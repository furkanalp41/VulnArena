package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/vulnarena/vulnarena/internal/server/middleware"
	"github.com/vulnarena/vulnarena/internal/service"
)

type APIKeyHandler struct {
	apiKeyService *service.APIKeyService
}

func NewAPIKeyHandler(apiKeyService *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{apiKeyService: apiKeyService}
}

func (h *APIKeyHandler) Generate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	rawKey, err := h.apiKeyService.GenerateAPIKey(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate api key")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"api_key": rawKey,
		"message": "Store this key securely. It will not be shown again.",
	})
}

func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.apiKeyService.RevokeAPIKey(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke api key")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "API key revoked successfully",
	})
}

func (h *APIKeyHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	info, err := h.apiKeyService.GetAPIKeyInfo(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if info == nil {
		writeError(w, http.StatusNotFound, "no API key found")
		return
	}

	writeJSON(w, http.StatusOK, info)
}
