package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omnichannel/lead_management_service/internal/domain"
	"github.com/omnichannel/lead_management_service/internal/service"
)

// LeadHandler handles all HTTP requests related to Leads.
type LeadHandler struct {
	leadService *service.LeadService
}

// NewLeadHandler creates a new LeadHandler.
func NewLeadHandler(svc *service.LeadService) *LeadHandler {
	return &LeadHandler{leadService: svc}
}

// RegisterRoutes connects our URL paths to our functions
func (h *LeadHandler) RegisterRoutes(r chi.Router) {
	// We group our routes under /api/v1/leads and apply our MockAuthMiddleware
	r.Route("/api/v1/leads", func(r chi.Router) {
		r.Use(MockAuthMiddleware)
		
		r.Get("/", h.ListLeads)
		r.Post("/", h.CreateLead)
		r.Patch("/{id}/stage", h.UpdateLeadStage)
		r.Patch("/{id}/assign", h.AssignLead)
		r.Patch("/{id}/tags", h.UpdateLeadTags)
		// We'll add notes and reminders later!
	})

	// Pipeline Stages API
	r.Route("/api/v1/stages", func(r chi.Router) {
		r.Use(MockAuthMiddleware)
		r.Get("/", h.ListWorkspaceStages)
		r.Post("/", h.CreateLeadStage)
	})
}

// ---- The Actual API Endpoints ----

// ListLeads runs when a user visits GET /api/v1/leads
func (h *LeadHandler) ListLeads(w http.ResponseWriter, r *http.Request) {
	// 1. Get the Workspace ID that our Middleware secretly attached to the request
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)

	// 2. Ask the Service Layer (The Head Chef) to get the leads
	leads, err := h.leadService.ListLeads(r.Context(), workspaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Convert the Go data into JSON and send it back to the web browser
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(leads)
}

// CreateLead runs when a user submits a form POST /api/v1/leads
func (h *LeadHandler) CreateLead(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)

	// 1. Read the JSON the web browser sent us
	var lead domain.Lead
	if err := json.NewDecoder(r.Body).Decode(&lead); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}
	
	// 2. Attach the Workspace ID for security
	lead.WorkspaceID = workspaceID

	// 3. Ask the Service Layer to create the lead (this will save it and fire a Redis event!)
	if err := h.leadService.CreateLead(r.Context(), &lead); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(lead)
}

// UpdateLeadStage runs when an agent drags-and-drops a lead (PATCH /api/v1/leads/123/stage)
func (h *LeadHandler) UpdateLeadStage(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)
	
	// 1. Get the Lead ID from the URL (e.g., the '123' in /leads/123/stage)
	leadID := chi.URLParam(r, "id")

	// 2. Read the JSON the web browser sent us to see what the new stage is
	var requestBody struct {
		NewStage string `json:"new_stage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	// 3. Ask the Service Layer to update the stage and trigger the AI summary
	err := h.leadService.UpdateLeadStage(r.Context(), workspaceID, leadID, requestBody.NewStage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 4. Tell the web browser it was a success!
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// AssignLead runs when an admin clicks "Assign to Bob" (PATCH /api/v1/leads/123/assign)
func (h *LeadHandler) AssignLead(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)
	leadID := chi.URLParam(r, "id")

	var requestBody struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	err := h.leadService.AssignLead(r.Context(), workspaceID, leadID, requestBody.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// UpdateLeadTags runs when an agent adds labels to a chat (PATCH /api/v1/leads/123/tags)
func (h *LeadHandler) UpdateLeadTags(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)
	leadID := chi.URLParam(r, "id")

	var requestBody struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	err := h.leadService.UpdateLeadTags(r.Context(), workspaceID, leadID, requestBody.Tags)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// ---- Pillar 1: Pipeline Configuration Endpoints ----

// CreateLeadStage runs when an admin submits a form POST /api/v1/stages
func (h *LeadHandler) CreateLeadStage(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)

	var stage domain.LeadStage
	if err := json.NewDecoder(r.Body).Decode(&stage); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}
	
	// Enforce Multi-Tenant Security
	stage.WorkspaceID = workspaceID

	if err := h.leadService.CreateLeadStage(r.Context(), &stage); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(stage)
}

// ListWorkspaceStages runs when the frontend fetches GET /api/v1/stages to render the Kanban board
func (h *LeadHandler) ListWorkspaceStages(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)

	stages, err := h.leadService.ListWorkspaceStages(r.Context(), workspaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stages)
}
