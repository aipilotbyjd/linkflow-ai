// Package features provides workflow folder management
package features

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Folder represents a workflow folder
type Folder struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ParentID    *string   `json:"parentId,omitempty"`
	UserID      string    `json:"userId"`
	WorkspaceID string    `json:"workspaceId"`
	Color       string    `json:"color"`
	Icon        string    `json:"icon"`
	Order       int       `json:"order"`
	Path        string    `json:"path"` // Full path like "/root/subfolder"
	Depth       int       `json:"depth"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// FolderRepository defines folder persistence
type FolderRepository interface {
	Create(ctx context.Context, folder *Folder) error
	FindByID(ctx context.Context, id string) (*Folder, error)
	Update(ctx context.Context, folder *Folder) error
	Delete(ctx context.Context, id string) error
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Folder, error)
	ListByParent(ctx context.Context, parentID string) ([]*Folder, error)
	ListRoot(ctx context.Context, workspaceID string) ([]*Folder, error)
	Move(ctx context.Context, folderID string, newParentID *string) error
}

// WorkflowFolder links workflows to folders
type WorkflowFolder struct {
	WorkflowID string    `json:"workflowId"`
	FolderID   string    `json:"folderId"`
	AddedAt    time.Time `json:"addedAt"`
}

// FolderService manages workflow folders
type FolderService struct {
	repo     FolderRepository
	maxDepth int
}

// NewFolderService creates a new folder service
func NewFolderService(repo FolderRepository) *FolderService {
	return &FolderService{
		repo:     repo,
		maxDepth: 10,
	}
}

// CreateFolder creates a new folder
func (s *FolderService) CreateFolder(ctx context.Context, folder *Folder) error {
	if folder.ID == "" {
		folder.ID = uuid.New().String()
	}
	folder.CreatedAt = time.Now()
	folder.UpdatedAt = time.Now()

	// Calculate path and depth
	if folder.ParentID != nil {
		parent, err := s.repo.FindByID(ctx, *folder.ParentID)
		if err != nil {
			return fmt.Errorf("parent folder not found: %w", err)
		}
		folder.Path = parent.Path + "/" + folder.Name
		folder.Depth = parent.Depth + 1
		
		if folder.Depth > s.maxDepth {
			return fmt.Errorf("maximum folder depth (%d) exceeded", s.maxDepth)
		}
	} else {
		folder.Path = "/" + folder.Name
		folder.Depth = 0
	}

	return s.repo.Create(ctx, folder)
}

// GetFolder retrieves a folder by ID
func (s *FolderService) GetFolder(ctx context.Context, id string) (*Folder, error) {
	return s.repo.FindByID(ctx, id)
}

// UpdateFolder updates a folder
func (s *FolderService) UpdateFolder(ctx context.Context, folder *Folder) error {
	folder.UpdatedAt = time.Now()
	
	// Recalculate path if name changed
	if folder.ParentID != nil {
		parent, err := s.repo.FindByID(ctx, *folder.ParentID)
		if err != nil {
			return err
		}
		folder.Path = parent.Path + "/" + folder.Name
	} else {
		folder.Path = "/" + folder.Name
	}

	return s.repo.Update(ctx, folder)
}

// DeleteFolder deletes a folder and optionally its contents
func (s *FolderService) DeleteFolder(ctx context.Context, id string, recursive bool) error {
	if recursive {
		// Delete all child folders first
		children, err := s.repo.ListByParent(ctx, id)
		if err != nil {
			return err
		}
		for _, child := range children {
			if err := s.DeleteFolder(ctx, child.ID, true); err != nil {
				return err
			}
		}
	} else {
		// Check if has children
		children, err := s.repo.ListByParent(ctx, id)
		if err != nil {
			return err
		}
		if len(children) > 0 {
			return fmt.Errorf("folder has subfolders, use recursive delete")
		}
	}

	return s.repo.Delete(ctx, id)
}

// MoveFolder moves a folder to a new parent
func (s *FolderService) MoveFolder(ctx context.Context, folderID string, newParentID *string) error {
	folder, err := s.repo.FindByID(ctx, folderID)
	if err != nil {
		return err
	}

	// Check for circular reference
	if newParentID != nil {
		current := newParentID
		for current != nil {
			if *current == folderID {
				return fmt.Errorf("cannot move folder to its own child")
			}
			parent, err := s.repo.FindByID(ctx, *current)
			if err != nil {
				break
			}
			current = parent.ParentID
		}
	}

	// Update depth and path
	if newParentID != nil {
		parent, err := s.repo.FindByID(ctx, *newParentID)
		if err != nil {
			return fmt.Errorf("new parent not found: %w", err)
		}
		folder.ParentID = newParentID
		folder.Path = parent.Path + "/" + folder.Name
		folder.Depth = parent.Depth + 1
	} else {
		folder.ParentID = nil
		folder.Path = "/" + folder.Name
		folder.Depth = 0
	}

	folder.UpdatedAt = time.Now()
	return s.repo.Update(ctx, folder)
}

// ListFolders lists all folders in a workspace
func (s *FolderService) ListFolders(ctx context.Context, workspaceID string) ([]*Folder, error) {
	return s.repo.ListByWorkspace(ctx, workspaceID)
}

// ListRootFolders lists root folders in a workspace
func (s *FolderService) ListRootFolders(ctx context.Context, workspaceID string) ([]*Folder, error) {
	return s.repo.ListRoot(ctx, workspaceID)
}

// ListSubfolders lists subfolders of a folder
func (s *FolderService) ListSubfolders(ctx context.Context, folderID string) ([]*Folder, error) {
	return s.repo.ListByParent(ctx, folderID)
}

// GetFolderTree returns the folder tree structure
func (s *FolderService) GetFolderTree(ctx context.Context, workspaceID string) (*FolderTree, error) {
	folders, err := s.repo.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	return buildTree(folders), nil
}

// FolderTree represents the folder hierarchy
type FolderTree struct {
	Folder   *Folder       `json:"folder,omitempty"`
	Children []*FolderTree `json:"children"`
}

func buildTree(folders []*Folder) *FolderTree {
	// Build map by ID
	folderMap := make(map[string]*FolderTree)
	for _, f := range folders {
		folderMap[f.ID] = &FolderTree{
			Folder:   f,
			Children: []*FolderTree{},
		}
	}

	// Build tree
	root := &FolderTree{Children: []*FolderTree{}}
	for _, f := range folders {
		node := folderMap[f.ID]
		if f.ParentID == nil {
			root.Children = append(root.Children, node)
		} else if parent, ok := folderMap[*f.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}

	return root
}

// InMemoryFolderRepository implements FolderRepository in memory
type InMemoryFolderRepository struct {
	folders map[string]*Folder
	mu      sync.RWMutex
}

// NewInMemoryFolderRepository creates a new in-memory folder repository
func NewInMemoryFolderRepository() *InMemoryFolderRepository {
	return &InMemoryFolderRepository{
		folders: make(map[string]*Folder),
	}
}

func (r *InMemoryFolderRepository) Create(ctx context.Context, folder *Folder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.folders[folder.ID] = folder
	return nil
}

func (r *InMemoryFolderRepository) FindByID(ctx context.Context, id string) (*Folder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	folder, ok := r.folders[id]
	if !ok {
		return nil, fmt.Errorf("folder not found")
	}
	return folder, nil
}

func (r *InMemoryFolderRepository) Update(ctx context.Context, folder *Folder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.folders[folder.ID] = folder
	return nil
}

func (r *InMemoryFolderRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.folders, id)
	return nil
}

func (r *InMemoryFolderRepository) ListByWorkspace(ctx context.Context, workspaceID string) ([]*Folder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var folders []*Folder
	for _, f := range r.folders {
		if f.WorkspaceID == workspaceID {
			folders = append(folders, f)
		}
	}
	return folders, nil
}

func (r *InMemoryFolderRepository) ListByParent(ctx context.Context, parentID string) ([]*Folder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var folders []*Folder
	for _, f := range r.folders {
		if f.ParentID != nil && *f.ParentID == parentID {
			folders = append(folders, f)
		}
	}
	return folders, nil
}

func (r *InMemoryFolderRepository) ListRoot(ctx context.Context, workspaceID string) ([]*Folder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var folders []*Folder
	for _, f := range r.folders {
		if f.WorkspaceID == workspaceID && f.ParentID == nil {
			folders = append(folders, f)
		}
	}
	return folders, nil
}

func (r *InMemoryFolderRepository) Move(ctx context.Context, folderID string, newParentID *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	folder, ok := r.folders[folderID]
	if !ok {
		return fmt.Errorf("folder not found")
	}
	folder.ParentID = newParentID
	return nil
}
