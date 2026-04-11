package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vulnarena/vulnarena/internal/server/middleware"
	"github.com/vulnarena/vulnarena/internal/service"
)

type CommunityHandler struct {
	communityService *service.CommunityService
}

func NewCommunityHandler(communityService *service.CommunityService) *CommunityHandler {
	return &CommunityHandler{communityService: communityService}
}

func (h *CommunityHandler) SubmitChallenge(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var input service.CommunitySubmitInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Title == "" || input.VulnerableCode == "" || input.TargetVulnerability == "" {
		writeError(w, http.StatusBadRequest, "title, vulnerable_code, and target_vulnerability are required")
		return
	}
	if input.Difficulty < 1 || input.Difficulty > 10 {
		writeError(w, http.StatusBadRequest, "difficulty must be between 1 and 10")
		return
	}
	if input.LanguageSlug == "" || input.VulnCategorySlug == "" {
		writeError(w, http.StatusBadRequest, "language_slug and vuln_category_slug are required")
		return
	}

	ch, err := h.communityService.SubmitChallenge(r.Context(), userID, input)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient XP") {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to submit challenge")
		return
	}

	writeJSON(w, http.StatusCreated, ch)
}

func (h *CommunityHandler) ListMyChallenges(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	challenges, total, err := h.communityService.ListMyChallenges(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list challenges")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"challenges": challenges,
		"total":      total,
	})
}

func (h *CommunityHandler) GetChallenge(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid challenge ID")
		return
	}

	ch, err := h.communityService.GetMyChallenge(r.Context(), userID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "challenge not found")
		return
	}

	writeJSON(w, http.StatusOK, ch)
}

func (h *CommunityHandler) UpdateChallenge(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid challenge ID")
		return
	}

	var input service.CommunitySubmitInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Title == "" || input.VulnerableCode == "" || input.TargetVulnerability == "" {
		writeError(w, http.StatusBadRequest, "title, vulnerable_code, and target_vulnerability are required")
		return
	}
	if input.Difficulty < 1 || input.Difficulty > 10 {
		writeError(w, http.StatusBadRequest, "difficulty must be between 1 and 10")
		return
	}

	ch, err := h.communityService.UpdateChallenge(r.Context(), userID, id, input)
	if err != nil {
		if strings.Contains(err.Error(), "pending") {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update challenge")
		return
	}

	writeJSON(w, http.StatusOK, ch)
}

func (h *CommunityHandler) DeleteChallenge(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid challenge ID")
		return
	}

	if err := h.communityService.DeleteChallenge(r.Context(), userID, id); err != nil {
		writeError(w, http.StatusNotFound, "challenge not found or not deletable")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
