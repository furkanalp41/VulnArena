package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vulnarena/vulnarena/internal/server/middleware"
	"github.com/vulnarena/vulnarena/internal/service"
)

type TeamHandler struct {
	teamService *service.TeamService
}

func NewTeamHandler(teamService *service.TeamService) *TeamHandler {
	return &TeamHandler{teamService: teamService}
}

func (h *TeamHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := h.teamService.ListTeams(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list teams")
		return
	}
	writeJSON(w, http.StatusOK, teams)
}

func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		writeError(w, http.StatusBadRequest, "missing team tag")
		return
	}

	team, err := h.teamService.GetTeam(r.Context(), tag)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get team")
		return
	}
	if team == nil {
		writeError(w, http.StatusNotFound, "team not found")
		return
	}

	writeJSON(w, http.StatusOK, team)
}

func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var input service.CreateTeamInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	team, err := h.teamService.CreateTeam(r.Context(), userID, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, team)
}

func (h *TeamHandler) JoinTeam(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		writeError(w, http.StatusBadRequest, "missing team tag")
		return
	}

	if err := h.teamService.JoinTeam(r.Context(), userID, tag); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "joined team"})
}

func (h *TeamHandler) LeaveTeam(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	if err := h.teamService.LeaveTeam(r.Context(), userID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "left team"})
}

func (h *TeamHandler) GetMyTeam(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	team, err := h.teamService.GetUserTeam(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get team")
		return
	}
	if team == nil {
		writeJSON(w, http.StatusOK, map[string]any{"team": nil})
		return
	}

	full, err := h.teamService.GetTeam(r.Context(), team.Tag)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get team details")
		return
	}

	writeJSON(w, http.StatusOK, full)
}

func (h *TeamHandler) GetTeamLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := h.teamService.GetTeamLeaderboard(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get team leaderboard")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}
