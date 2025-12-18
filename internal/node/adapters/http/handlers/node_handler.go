package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/node/adapters/http/dto"
	"github.com/linkflow-ai/linkflow-ai/internal/node/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/node/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
)

// NodeHandler handles HTTP requests for node definitions
type NodeHandler struct {
	service *service.NodeService
	logger  logger.Logger
}

// NewNodeHandler creates a new node handler
func NewNodeHandler(service *service.NodeService, logger logger.Logger) *NodeHandler {
	return &NodeHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers node routes
func (h *NodeHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/nodes", h.CreateNode).Methods("POST")
	router.HandleFunc("/nodes", h.ListNodes).Methods("GET")
	router.HandleFunc("/nodes/{id}", h.GetNode).Methods("GET")
	router.HandleFunc("/nodes/{id}", h.UpdateNode).Methods("PUT")
	router.HandleFunc("/nodes/{id}", h.DeleteNode).Methods("DELETE")
	router.HandleFunc("/nodes/{id}/clone", h.CloneNode).Methods("POST")
	router.HandleFunc("/nodes/categories", h.ListCategories).Methods("GET")
	router.HandleFunc("/nodes/types", h.ListTypes).Methods("GET")
}

// CreateNode creates a new node definition
func (h *NodeHandler) CreateNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req dto.CreateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create node
	node, err := h.service.CreateNode(ctx, service.CreateNodeCommand{
		Name:             req.Name,
		Type:             req.Type,
		Category:         req.Category,
		Description:      req.Description,
		Icon:             req.Icon,
		Color:            req.Color,
		Inputs:           h.convertPortDTOsToModel(req.Inputs),
		Outputs:          h.convertPortDTOsToModel(req.Outputs),
		Properties:       h.convertPropertyDTOsToModel(req.Properties),
		ExecutionHandler: req.ExecutionHandler,
		Documentation:    req.Documentation,
		Tags:             req.Tags,
		IsPremium:        req.IsPremium,
	})
	if err != nil {
		if err == service.ErrNodeAlreadyExists {
			h.respondError(w, http.StatusConflict, "node with this name already exists")
			return
		}
		h.logger.Error("Failed to create node", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to create node")
		return
	}

	// Convert to response DTO
	resp := h.nodeToDTO(node)
	h.respondJSON(w, http.StatusCreated, resp)
}

// GetNode gets a node by ID
func (h *NodeHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get node ID from path
	vars := mux.Vars(r)
	nodeID := vars["id"]

	// Get node
	node, err := h.service.GetNode(ctx, model.NodeID(nodeID))
	if err != nil {
		if err == service.ErrNodeNotFound {
			h.respondError(w, http.StatusNotFound, "node not found")
			return
		}
		h.logger.Error("Failed to get node", "error", err, "node_id", nodeID)
		h.respondError(w, http.StatusInternalServerError, "failed to get node")
		return
	}

	// Convert to response DTO
	resp := h.nodeToDTO(node)
	h.respondJSON(w, http.StatusOK, resp)
}

// UpdateNode updates a node definition
func (h *NodeHandler) UpdateNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get node ID from path
	vars := mux.Vars(r)
	nodeID := vars["id"]

	// Parse request
	var req dto.UpdateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update node
	node, err := h.service.UpdateNode(ctx, service.UpdateNodeCommand{
		ID:               nodeID,
		Name:             req.Name,
		Description:      req.Description,
		Icon:             req.Icon,
		Color:            req.Color,
		Inputs:           h.convertPortDTOsToModel(req.Inputs),
		Outputs:          h.convertPortDTOsToModel(req.Outputs),
		Properties:       h.convertPropertyDTOsToModel(req.Properties),
		ExecutionHandler: req.ExecutionHandler,
		Documentation:    req.Documentation,
		Tags:             req.Tags,
		Status:           req.Status,
	})
	if err != nil {
		if err == service.ErrNodeNotFound {
			h.respondError(w, http.StatusNotFound, "node not found")
			return
		}
		if err == service.ErrSystemNode {
			h.respondError(w, http.StatusForbidden, "cannot modify system node")
			return
		}
		h.logger.Error("Failed to update node", "error", err, "node_id", nodeID)
		h.respondError(w, http.StatusInternalServerError, "failed to update node")
		return
	}

	// Convert to response DTO
	resp := h.nodeToDTO(node)
	h.respondJSON(w, http.StatusOK, resp)
}

// DeleteNode deletes a node definition
func (h *NodeHandler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get node ID from path
	vars := mux.Vars(r)
	nodeID := vars["id"]

	// Delete node
	err := h.service.DeleteNode(ctx, model.NodeID(nodeID))
	if err != nil {
		if err == service.ErrNodeNotFound {
			h.respondError(w, http.StatusNotFound, "node not found")
			return
		}
		if err == service.ErrSystemNode {
			h.respondError(w, http.StatusForbidden, "cannot delete system node")
			return
		}
		h.logger.Error("Failed to delete node", "error", err, "node_id", nodeID)
		h.respondError(w, http.StatusInternalServerError, "failed to delete node")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "node deleted successfully"})
}

// CloneNode clones a node definition
func (h *NodeHandler) CloneNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get node ID from path
	vars := mux.Vars(r)
	nodeID := vars["id"]

	// Clone node
	node, err := h.service.CloneNode(ctx, model.NodeID(nodeID))
	if err != nil {
		if err == service.ErrNodeNotFound {
			h.respondError(w, http.StatusNotFound, "node not found")
			return
		}
		h.logger.Error("Failed to clone node", "error", err, "node_id", nodeID)
		h.respondError(w, http.StatusInternalServerError, "failed to clone node")
		return
	}

	// Convert to response DTO
	resp := h.nodeToDTO(node)
	h.respondJSON(w, http.StatusCreated, resp)
}

// ListNodes lists node definitions
func (h *NodeHandler) ListNodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	query := r.URL.Query()
	offset, _ := strconv.Atoi(query.Get("offset"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	
	if limit == 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	// Parse boolean parameters
	var isSystem, isPremium *bool
	if sys := query.Get("isSystem"); sys != "" {
		val := sys == "true"
		isSystem = &val
	}
	if prem := query.Get("isPremium"); prem != "" {
		val := prem == "true"
		isPremium = &val
	}

	// List nodes
	nodes, total, err := h.service.ListNodes(ctx, service.ListNodesQuery{
		Type:        query.Get("type"),
		Category:    query.Get("category"),
		Status:      query.Get("status"),
		IsSystem:    isSystem,
		IsPremium:   isPremium,
		SearchQuery: query.Get("q"),
		Offset:      offset,
		Limit:       limit,
	})
	if err != nil {
		h.logger.Error("Failed to list nodes", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to list nodes")
		return
	}

	// Convert to response DTOs
	items := make([]dto.NodeResponse, len(nodes))
	for i, node := range nodes {
		items[i] = h.nodeToDTO(node)
	}

	resp := dto.ListNodesResponse{
		Items: items,
		Total: total,
		Pagination: dto.Pagination{
			Offset: offset,
			Limit:  limit,
			Total:  total,
		},
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// ListCategories lists available node categories
func (h *NodeHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories := []string{
		string(model.NodeCategoryCore),
		string(model.NodeCategoryIntegration),
		string(model.NodeCategoryTransform),
		string(model.NodeCategoryControl),
		string(model.NodeCategoryCommunication),
		string(model.NodeCategoryStorage),
		string(model.NodeCategoryCustom),
	}
	
	h.respondJSON(w, http.StatusOK, categories)
}

// ListTypes lists available node types
func (h *NodeHandler) ListTypes(w http.ResponseWriter, r *http.Request) {
	types := []string{
		string(model.NodeTypeTrigger),
		string(model.NodeTypeAction),
		string(model.NodeTypeCondition),
		string(model.NodeTypeLoop),
		string(model.NodeTypeSwitch),
		string(model.NodeTypeTransform),
		string(model.NodeTypeAggregator),
		string(model.NodeTypeSchedule),
		string(model.NodeTypeWebhook),
		string(model.NodeTypeHTTP),
		string(model.NodeTypeDatabase),
		string(model.NodeTypeFile),
		string(model.NodeTypeEmail),
		string(model.NodeTypeNotification),
		string(model.NodeTypeCustom),
	}
	
	h.respondJSON(w, http.StatusOK, types)
}

// Helper methods

func (h *NodeHandler) nodeToDTO(node *model.NodeDefinition) dto.NodeResponse {
	return dto.NodeResponse{
		ID:               node.ID().String(),
		Name:             node.Name(),
		Type:             string(node.Type()),
		Category:         string(node.Category()),
		Description:      node.Description(),
		Icon:             node.Icon(),
		Color:            node.Color(),
		Version:          node.Version(),
		Status:           string(node.Status()),
		Inputs:           h.convertPortsToDTO(node.Inputs()),
		Outputs:          h.convertPortsToDTO(node.Outputs()),
		Properties:       h.convertPropertiesToDTO(node.Properties()),
		Tags:             node.Tags(),
		IsSystem:         node.IsSystem(),
		IsPremium:        node.IsPremium(),
		ExecutionHandler: node.ExecutionHandler(),
		CreatedAt:        node.CreatedAt(),
		UpdatedAt:        node.UpdatedAt(),
	}
}

func (h *NodeHandler) convertPortsToDTO(ports []model.NodePort) []dto.NodePort {
	result := make([]dto.NodePort, len(ports))
	for i, port := range ports {
		result[i] = dto.NodePort{
			ID:           port.ID,
			Name:         port.Name,
			Type:         port.Type,
			Required:     port.Required,
			Multiple:     port.Multiple,
			Description:  port.Description,
			DefaultValue: port.DefaultValue,
			Schema:       port.Schema,
		}
	}
	return result
}

func (h *NodeHandler) convertPropertiesToDTO(properties []model.NodeProperty) []dto.NodeProperty {
	result := make([]dto.NodeProperty, len(properties))
	for i, prop := range properties {
		options := make([]dto.PropertyOption, len(prop.Options))
		for j, opt := range prop.Options {
			options[j] = dto.PropertyOption{
				Label: opt.Label,
				Value: opt.Value,
			}
		}
		
		result[i] = dto.NodeProperty{
			ID:           prop.ID,
			Name:         prop.Name,
			Type:         prop.Type,
			Required:     prop.Required,
			Description:  prop.Description,
			DefaultValue: prop.DefaultValue,
			Options:      options,
			Validation:   prop.Validation,
			Placeholder:  prop.Placeholder,
			Hidden:       prop.Hidden,
		}
	}
	return result
}

func (h *NodeHandler) convertPortDTOsToModel(ports []dto.NodePort) []model.NodePort {
	result := make([]model.NodePort, len(ports))
	for i, port := range ports {
		result[i] = model.NodePort{
			ID:           port.ID,
			Name:         port.Name,
			Type:         port.Type,
			Required:     port.Required,
			Multiple:     port.Multiple,
			Description:  port.Description,
			DefaultValue: port.DefaultValue,
			Schema:       port.Schema,
		}
	}
	return result
}

func (h *NodeHandler) convertPropertyDTOsToModel(properties []dto.NodeProperty) []model.NodeProperty {
	result := make([]model.NodeProperty, len(properties))
	for i, prop := range properties {
		options := make([]model.PropertyOption, len(prop.Options))
		for j, opt := range prop.Options {
			options[j] = model.PropertyOption{
				Label: opt.Label,
				Value: opt.Value,
			}
		}
		
		result[i] = model.NodeProperty{
			ID:           prop.ID,
			Name:         prop.Name,
			Type:         prop.Type,
			Required:     prop.Required,
			Description:  prop.Description,
			DefaultValue: prop.DefaultValue,
			Options:      options,
			Validation:   prop.Validation,
			Placeholder:  prop.Placeholder,
			Hidden:       prop.Hidden,
		}
	}
	return result
}

func (h *NodeHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

func (h *NodeHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
