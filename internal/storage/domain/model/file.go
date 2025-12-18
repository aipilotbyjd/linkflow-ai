package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type FileID string

func NewFileID() FileID {
	return FileID(uuid.New().String())
}

type FileType string

const (
	FileTypeImage    FileType = "image"
	FileTypeDocument FileType = "document"
	FileTypeVideo    FileType = "video"
	FileTypeAudio    FileType = "audio"
	FileTypeArchive  FileType = "archive"
	FileTypeOther    FileType = "other"
)

type File struct {
	id          FileID
	userID      string
	name        string
	path        string
	mimeType    string
	fileType    FileType
	size        int64
	checksum    string
	metadata    map[string]interface{}
	tags        []string
	isPublic    bool
	uploadedAt  time.Time
	modifiedAt  time.Time
	accessCount int
	lastAccess  time.Time
	expiresAt   *time.Time
}

func NewFile(userID, name, path string, size int64) (*File, error) {
	if userID == "" || name == "" || path == "" {
		return nil, errors.New("userID, name, and path are required")
	}

	if size < 0 {
		return nil, errors.New("file size must be non-negative")
	}

	now := time.Now()
	return &File{
		id:         NewFileID(),
		userID:     userID,
		name:       name,
		path:       path,
		size:       size,
		fileType:   FileTypeOther,
		metadata:   make(map[string]interface{}),
		tags:       []string{},
		uploadedAt: now,
		modifiedAt: now,
		isPublic:   false,
	}, nil
}

func (f *File) ID() FileID                { return f.id }
func (f *File) UserID() string            { return f.userID }
func (f *File) Name() string              { return f.name }
func (f *File) Path() string              { return f.path }
func (f *File) Size() int64               { return f.size }
func (f *File) IsPublic() bool            { return f.isPublic }
func (f *File) UploadedAt() time.Time     { return f.uploadedAt }
func (f *File) Tags() []string            { return f.tags }
func (f *File) Metadata() map[string]interface{} { return f.metadata }

func (f *File) SetMimeType(mimeType string) {
	f.mimeType = mimeType
	f.fileType = inferFileType(mimeType)
}

func (f *File) SetChecksum(checksum string) {
	f.checksum = checksum
}

func (f *File) SetPublic(public bool) {
	f.isPublic = public
	f.modifiedAt = time.Now()
}

func (f *File) AddTag(tag string) {
	f.tags = append(f.tags, tag)
	f.modifiedAt = time.Now()
}

func (f *File) SetMetadata(key string, value interface{}) {
	f.metadata[key] = value
	f.modifiedAt = time.Now()
}

func (f *File) RecordAccess() {
	f.accessCount++
	f.lastAccess = time.Now()
}

func inferFileType(mimeType string) FileType {
	switch {
	case contains(mimeType, "image"):
		return FileTypeImage
	case contains(mimeType, "video"):
		return FileTypeVideo
	case contains(mimeType, "audio"):
		return FileTypeAudio
	case contains(mimeType, "pdf"), contains(mimeType, "document"), contains(mimeType, "text"):
		return FileTypeDocument
	case contains(mimeType, "zip"), contains(mimeType, "tar"), contains(mimeType, "archive"):
		return FileTypeArchive
	default:
		return FileTypeOther
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
