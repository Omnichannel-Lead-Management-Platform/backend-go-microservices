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
		// We'll add notes and reminders later!
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
