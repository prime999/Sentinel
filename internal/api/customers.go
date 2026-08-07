package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

type customerRequest struct {
	Name         string `json:"name"`
	MonitorQuota *int   `json:"monitor_quota"`
}

func (s *Server) handleListCustomers(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListCustomers()
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if items == nil {
		jsonOK(w, []any{})
		return
	}
	jsonOK(w, items)
}

func (s *Server) handleCreateCustomer(w http.ResponseWriter, r *http.Request) {
	var req customerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	quota := 1
	if req.MonitorQuota != nil {
		quota = *req.MonitorQuota
	}
	c, err := s.store.CreateCustomer(req.Name, quota)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = s.store.InsertAudit(currentUser(r).Username, "create", "customer", c.Name)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, c)
}

func (s *Server) handleUpdateCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req customerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	existing, err := s.store.GetCustomer(id)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if existing == nil {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = existing.Name
	}
	quota := existing.MonitorQuota
	if req.MonitorQuota != nil {
		quota = *req.MonitorQuota
	}
	c, err := s.store.UpdateCustomer(id, name, quota)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = s.store.InsertAudit(currentUser(r).Username, "update", "customer", c.Name)
	jsonOK(w, c)
}

func (s *Server) handleDeleteCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := s.store.GetCustomer(id)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if existing == nil {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	if err := s.store.DeleteCustomer(id); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = s.store.InsertAudit(currentUser(r).Username, "delete", "customer", existing.Name)
	w.WriteHeader(http.StatusNoContent)
}
