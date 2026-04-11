package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vulnarena/vulnarena/internal/model"
	"github.com/vulnarena/vulnarena/internal/repository"
	"github.com/vulnarena/vulnarena/internal/service"
)

type AcademyHandler struct {
	academyService *service.AcademyService
}

func NewAcademyHandler(academyService *service.AcademyService) *AcademyHandler {
	return &AcademyHandler{academyService: academyService}
}

func (h *AcademyHandler) ListLessons(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := model.LessonFilter{
		Category: q.Get("category"),
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

	lessons, total, err := h.academyService.ListLessons(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list lessons")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"lessons": lessons,
		"total":   total,
	})
}

func (h *AcademyHandler) GetLesson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid lesson ID")
		return
	}

	lesson, err := h.academyService.GetLesson(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "lesson not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get lesson")
		return
	}

	writeJSON(w, http.StatusOK, lesson)
}
