package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/storage/domain/model"
)

type StorageService struct {
	baseDir string
	logger  logger.Logger
	files   map[string]*model.File // In-memory storage for demo
}

func NewStorageService(baseDir string, logger logger.Logger) *StorageService {
	if baseDir == "" {
		baseDir = "/tmp/linkflow-storage"
	}
	
	// Create base directory if it doesn't exist
	os.MkdirAll(baseDir, 0755)
	
	return &StorageService{
		baseDir: baseDir,
		logger:  logger,
		files:   make(map[string]*model.File),
	}
}

type UploadFileCommand struct {
	UserID   string
	FileName string
	Reader   io.Reader
	Size     int64
	MimeType string
	Tags     []string
	Metadata map[string]interface{}
}

func (s *StorageService) UploadFile(ctx context.Context, cmd UploadFileCommand) (*model.File, error) {
	// Generate unique file path
	fileID := model.NewFileID()
	userDir := filepath.Join(s.baseDir, cmd.UserID)
	os.MkdirAll(userDir, 0755)
	
	filePath := filepath.Join(userDir, string(fileID))
	
	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	// Calculate checksum while copying
	hash := md5.New()
	writer := io.MultiWriter(file, hash)
	
	written, err := io.Copy(writer, cmd.Reader)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	
	// Create file model
	fileModel, err := model.NewFile(cmd.UserID, cmd.FileName, filePath, written)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to create file model: %w", err)
	}
	
	// Set additional properties
	fileModel.SetMimeType(cmd.MimeType)
	fileModel.SetChecksum(hex.EncodeToString(hash.Sum(nil)))
	
	for _, tag := range cmd.Tags {
		fileModel.AddTag(tag)
	}
	
	for key, value := range cmd.Metadata {
		fileModel.SetMetadata(key, value)
	}
	
	// Store in memory (would be database in production)
	s.files[string(fileModel.ID())] = fileModel
	
	s.logger.Info("File uploaded",
		"file_id", fileModel.ID(),
		"user_id", cmd.UserID,
		"name", cmd.FileName,
		"size", written,
	)
	
	return fileModel, nil
}

func (s *StorageService) DownloadFile(ctx context.Context, fileID string, userID string) (io.ReadCloser, *model.File, error) {
	// Get file metadata
	fileModel, exists := s.files[fileID]
	if !exists {
		return nil, nil, fmt.Errorf("file not found")
	}
	
	// Check ownership
	if fileModel.UserID() != userID && !fileModel.IsPublic() {
		return nil, nil, fmt.Errorf("access denied")
	}
	
	// Open file
	file, err := os.Open(fileModel.Path())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	// Record access
	fileModel.RecordAccess()
	
	s.logger.Debug("File downloaded",
		"file_id", fileID,
		"user_id", userID,
	)
	
	return file, fileModel, nil
}

func (s *StorageService) DeleteFile(ctx context.Context, fileID string, userID string) error {
	// Get file metadata
	fileModel, exists := s.files[fileID]
	if !exists {
		return fmt.Errorf("file not found")
	}
	
	// Check ownership
	if fileModel.UserID() != userID {
		return fmt.Errorf("access denied")
	}
	
	// Delete physical file
	if err := os.Remove(fileModel.Path()); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	
	// Remove from memory
	delete(s.files, fileID)
	
	s.logger.Info("File deleted",
		"file_id", fileID,
		"user_id", userID,
	)
	
	return nil
}

func (s *StorageService) ListFiles(ctx context.Context, userID string) ([]*model.File, error) {
	var files []*model.File
	
	for _, file := range s.files {
		if file.UserID() == userID || file.IsPublic() {
			files = append(files, file)
		}
	}
	
	return files, nil
}

func (s *StorageService) ShareFile(ctx context.Context, fileID string, userID string, public bool) error {
	fileModel, exists := s.files[fileID]
	if !exists {
		return fmt.Errorf("file not found")
	}
	
	// Check ownership
	if fileModel.UserID() != userID {
		return fmt.Errorf("access denied")
	}
	
	fileModel.SetPublic(public)
	
	s.logger.Info("File sharing updated",
		"file_id", fileID,
		"public", public,
	)
	
	return nil
}

func (s *StorageService) GetStorageStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	var totalSize int64
	var fileCount int
	var typeBreakdown = make(map[string]int)
	
	for _, file := range s.files {
		if file.UserID() == userID {
			totalSize += file.Size()
			fileCount++
			typeBreakdown[string(file.Metadata()["type"].(model.FileType))]++
		}
	}
	
	return map[string]interface{}{
		"total_size":     totalSize,
		"file_count":     fileCount,
		"type_breakdown": typeBreakdown,
		"quota_used":     float64(totalSize) / (1024 * 1024 * 1024), // GB
		"quota_limit":    10.0, // 10 GB limit
	}, nil
}

func (s *StorageService) CleanupExpiredFiles(ctx context.Context) error {
	now := time.Now()
	var deleted int
	
	for id, file := range s.files {
		if file.Metadata()["expires_at"] != nil {
			expiresAt := file.Metadata()["expires_at"].(time.Time)
			if now.After(expiresAt) {
				os.Remove(file.Path())
				delete(s.files, id)
				deleted++
			}
		}
	}
	
	s.logger.Info("Cleaned up expired files", "deleted", deleted)
	return nil
}
