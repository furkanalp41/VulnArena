package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vulnarena/vulnarena/internal/repository"
	"github.com/vulnarena/vulnarena/internal/server/middleware"
	"github.com/vulnarena/vulnarena/internal/service"
)

type AdminHandler struct {
	adminService *service.AdminService
}

func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (h *AdminHandler) GetPlatformStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.adminService.GetPlatformStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load platform stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *AdminHandler) CreateChallenge(w http.ResponseWriter, r *http.Request) {
	var input service.CreateChallengeInput
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

	ch, err := h.adminService.CreateChallenge(r.Context(), input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create challenge")
		return
	}

	writeJSON(w, http.StatusCreated, ch)
}

func (h *AdminHandler) UpdateChallenge(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid challenge ID")
		return
	}

	var input service.UpdateChallengeInput
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

	ch, err := h.adminService.UpdateChallenge(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "challenge not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update challenge")
		return
	}

	writeJSON(w, http.StatusOK, ch)
}

func (h *AdminHandler) CreateLesson(w http.ResponseWriter, r *http.Request) {
	var input service.CreateLessonInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Title == "" || input.Content == "" || input.Category == "" {
		writeError(w, http.StatusBadRequest, "title, content, and category are required")
		return
	}
	if input.Difficulty < 1 || input.Difficulty > 10 {
		writeError(w, http.StatusBadRequest, "difficulty must be between 1 and 10")
		return
	}

	lesson, err := h.adminService.CreateLesson(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create lesson")
		return
	}

	writeJSON(w, http.StatusCreated, lesson)
}

func (h *AdminHandler) ListCommunityQueue(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}

	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	limit := 20
	offset := (page - 1) * limit

	challenges, total, err := h.adminService.ListCommunityQueue(r.Context(), status, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list community queue")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"challenges": challenges,
		"total":      total,
		"page":       page,
	})
}

func (h *AdminHandler) GetCommunityChallenge(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid challenge ID")
		return
	}

	ch, err := h.adminService.GetCommunityChallenge(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "community challenge not found")
		return
	}

	writeJSON(w, http.StatusOK, ch)
}

func (h *AdminHandler) ReviewCommunityChallenge(w http.ResponseWriter, r *http.Request) {
	reviewerID := middleware.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid challenge ID")
		return
	}

	var input struct {
		Action string `json:"action"` // "approve" or "reject"
		Notes  string `json:"notes"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Action != "approve" && input.Action != "reject" {
		writeError(w, http.StatusBadRequest, "action must be 'approve' or 'reject'")
		return
	}

	if err := h.adminService.ReviewCommunityChallenge(r.Context(), id, reviewerID, input.Action, input.Notes); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to review challenge")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "reviewed", "action": input.Action})
}

func (h *AdminHandler) PublishCommunityChallenge(w http.ResponseWriter, r *http.Request) {
	reviewerID := middleware.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid challenge ID")
		return
	}

	challenge, err := h.adminService.PublishCommunityChallenge(r.Context(), id, reviewerID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "community challenge not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to publish challenge: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, challenge)
}
