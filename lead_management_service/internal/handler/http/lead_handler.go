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
		
		// Pillar 3: Internal Notes
		r.Get("/{id}/notes", h.GetInternalNotes)
		r.Post("/{id}/notes", h.AddInternalNote)
		
		// Chat History
		r.Get("/{id}/messages", h.GetMessagesByLead)
	})

	// Pipeline Stages API
	r.Route("/api/v1/stages", func(r chi.Router) {
		r.Use(MockAuthMiddleware)
		r.Get("/", h.ListWorkspaceStages)
		r.Post("/", h.CreateLeadStage)
		r.Patch("/{id}", h.UpdateLeadStageConfig)
		r.Delete("/{id}", h.DeleteLeadStage)
		r.Put("/reorder", h.ReorderLeadStages)
	})

	// Pillar 4: Message Templates API
	r.Route("/api/v1/templates", func(r chi.Router) {
		r.Use(MockAuthMiddleware)
		r.Get("/", h.GetMessageTemplates)
		r.Post("/", h.CreateMessageTemplate)
		r.Put("/{id}", h.UpdateMessageTemplate)
		r.Delete("/{id}", h.DeleteMessageTemplate)
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

// ---- Pillar 3: Internal Notes Endpoints ----

// GetInternalNotes fetches all private notes for a specific lead
func (h *LeadHandler) GetInternalNotes(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)
	leadID := chi.URLParam(r, "id")

	notes, err := h.leadService.GetInternalNotesByLead(r.Context(), workspaceID, leadID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

// GetMessagesByLead handles GET /api/v1/leads/{id}/messages
func (h *LeadHandler) GetMessagesByLead(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)
	leadID := chi.URLParam(r, "id")

	messages, err := h.leadService.GetMessagesByLead(r.Context(), workspaceID, leadID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// AddInternalNote allows an agent to leave a private note on a lead's conversation
func (h *LeadHandler) AddInternalNote(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)
	leadID := chi.URLParam(r, "id")

	var note domain.InternalNote
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}
	
	// Security & Context Rules
	note.WorkspaceID = workspaceID
	// In this simplified version, we'll pretend the lead ID is the conversation ID,
	// or that the frontend sends conversation_id in the JSON. If it's empty, we set it.
	if note.ConversationID == "" {
		note.ConversationID = leadID 
	}
	// The user ID would normally come from the Auth Middleware JWT
	note.UserID = r.Header.Get("X-User-ID") 

	if err := h.leadService.AddInternalNote(r.Context(), &note); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
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

// UpdateLeadStageConfig handles PATCH /api/v1/stages/{id}
func (h *LeadHandler) UpdateLeadStageConfig(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)
	stageID := chi.URLParam(r, "id")

	var requestBody struct {
		Label string `json:"label"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	stage := &domain.LeadStage{
		ID:          stageID,
		WorkspaceID: workspaceID,
		Label:       requestBody.Label,
		Color:       requestBody.Color,
	}

	if err := h.leadService.UpdateLeadStageConfig(r.Context(), stage); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// DeleteLeadStage handles DELETE /api/v1/stages/{id}
func (h *LeadHandler) DeleteLeadStage(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)
	stageID := chi.URLParam(r, "id")

	if err := h.leadService.DeleteLeadStage(r.Context(), workspaceID, stageID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// ReorderLeadStages handles PUT /api/v1/stages/reorder
func (h *LeadHandler) ReorderLeadStages(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)

	var requestBody struct {
		StageIDs []string `json:"stage_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	if err := h.leadService.ReorderLeadStages(r.Context(), workspaceID, requestBody.StageIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// ---- Pillar 4: Message Templates Endpoints ----

// CreateMessageTemplate handles POST /api/v1/templates
func (h *LeadHandler) CreateMessageTemplate(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)

	var template domain.MessageTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}
	template.WorkspaceID = workspaceID

	if err := h.leadService.CreateMessageTemplate(r.Context(), &template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

// GetMessageTemplates handles GET /api/v1/templates
func (h *LeadHandler) GetMessageTemplates(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)

	templates, err := h.leadService.GetMessageTemplates(r.Context(), workspaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// UpdateMessageTemplate handles PUT /api/v1/templates/{id}
func (h *LeadHandler) UpdateMessageTemplate(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)
	templateID := chi.URLParam(r, "id")

	var template domain.MessageTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}
	template.WorkspaceID = workspaceID
	template.ID = templateID

	if err := h.leadService.UpdateMessageTemplate(r.Context(), &template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// DeleteMessageTemplate handles DELETE /api/v1/templates/{id}
func (h *LeadHandler) DeleteMessageTemplate(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Context().Value(WorkspaceIDKey).(string)
	templateID := chi.URLParam(r, "id")

	if err := h.leadService.DeleteMessageTemplate(r.Context(), workspaceID, templateID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}
