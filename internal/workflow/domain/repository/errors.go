package repository

import "errors"

var (
	// ErrNotFound is returned when a workflow is not found
	ErrNotFound = errors.New("workflow not found")
	
	// ErrOptimisticLock is returned when optimistic locking fails
	ErrOptimisticLock = errors.New("optimistic lock: workflow was modified by another process")
	
	// ErrDuplicateName is returned when a workflow with the same name already exists
	ErrDuplicateName = errors.New("workflow with this name already exists")
)
