package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

const maxLogoBytes = 512 * 1024

func (s *Server) handleGetGeneral(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.store.GetOrgSettings()
	if err != nil {
		jsonInternal(w, err)
		return
	}
	jsonOK(w, cfg)
}

func (s *Server) handlePutGeneral(w http.ResponseWriter, r *http.Request) {
	var input models.OrgSettings
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}

	input.CompanyName = strings.TrimSpace(input.CompanyName)
	input.Tagline = strings.TrimSpace(input.Tagline)
	input.Logo = strings.TrimSpace(input.Logo)

	if input.Logo != "" {
		if len(input.Logo) > maxLogoBytes*4/3+1024 {
			jsonError(w, http.StatusBadRequest, "logo too large (max 512KB)")
			return
		}
		if !strings.HasPrefix(input.Logo, "data:image/") {
			jsonError(w, http.StatusBadRequest, "logo must be an image data URL")
			return
		}
	}

	if err := s.store.SaveOrgSettings(input); err != nil {
		jsonInternal(w, err)
		return
	}
	jsonOK(w, input)
}

func (s *Server) handleResetGeneral(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteSetting("org"); err != nil {
		jsonInternal(w, err)
		return
	}
	jsonOK(w, models.OrgSettings{})
}
