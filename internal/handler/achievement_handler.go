package handler

import (
	"net/http"

	"github.com/vulnarena/vulnarena/internal/service"
)

type AchievementHandler struct {
	achievementService *service.AchievementService
}

func NewAchievementHandler(achievementService *service.AchievementService) *AchievementHandler {
	return &AchievementHandler{achievementService: achievementService}
}

func (h *AchievementHandler) ListAchievements(w http.ResponseWriter, r *http.Request) {
	achievements, err := h.achievementService.GetAllAchievements(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load achievements")
		return
	}
	writeJSON(w, http.StatusOK, achievements)
}
