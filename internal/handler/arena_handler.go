package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vulnarena/vulnarena/internal/model"
	"github.com/vulnarena/vulnarena/internal/repository"
	"github.com/vulnarena/vulnarena/internal/server/middleware"
	"github.com/vulnarena/vulnarena/internal/service"
)

type ArenaHandler struct {
	arenaService *service.ArenaService
}

func NewArenaHandler(arenaService *service.ArenaService) *ArenaHandler {
	return &ArenaHandler{arenaService: arenaService}
}

func (h *ArenaHandler) ListChallenges(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := model.ChallengeFilter{
		LanguageSlug:     q.Get("language"),
		VulnCategorySlug: q.Get("category"),
	}

	if v := q.Get("difficulty_min"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.DifficultyMin = n
		}
	}
	if v := q.Get("difficulty_max"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.DifficultyMax = n
		}
	}
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Page = n
		}
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Limit = n
		}
	}

	challenges, total, err := h.arenaService.ListChallenges(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list challenges")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"challenges": challenges,
		"total":      total,
		"page":       filter.Page,
		"limit":      filter.Limit,
	})
}

func (h *ArenaHandler) GetChallenge(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid challenge ID")
		return
	}

	challenge, err := h.arenaService.GetChallenge(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "challenge not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get challenge")
		return
	}

	// Include user progress if authenticated
	userID := middleware.UserIDFromContext(r.Context())
	var progress *model.UserChallengeProgress
	if userID != uuid.Nil {
		progress, _ = h.arenaService.GetUserProgress(r.Context(), userID, id)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"challenge": challenge,
		"progress":  progress,
	})
}

func (h *ArenaHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	challengeID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid challenge ID")
		return
	}

	var input service.SubmitAnswerInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.AnswerText == "" {
		writeError(w, http.StatusBadRequest, "answer_text is required")
		return
	}

	if len(input.AnswerText) < 10 {
		writeError(w, http.StatusBadRequest, "answer must be at least 10 characters")
		return
	}

	result, err := h.arenaService.SubmitAnswer(r.Context(), userID, challengeID, input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "evaluation failed")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ArenaHandler) RevealSolution(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	challengeID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid challenge ID")
		return
	}

	result, err := h.arenaService.RevealSolution(r.Context(), userID, challengeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reveal solution")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ArenaHandler) GetSubmissionHistory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	challengeID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid challenge ID")
		return
	}

	subs, err := h.arenaService.GetSubmissionHistory(r.Context(), userID, challengeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get submissions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"submissions": subs,
	})
}
