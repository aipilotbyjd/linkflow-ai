package events

import "time"

// Integration Events
type IntegrationCreatedEvent struct {
	IntegrationID  string    `json:"integration_id"`
	UserID         string    `json:"user_id"`
	OrganizationID string    `json:"organization_id,omitempty"`
	Type           string    `json:"type"`
	Name           string    `json:"name"`
	Timestamp      time.Time `json:"timestamp"`
}

type IntegrationUpdatedEvent struct {
	IntegrationID string                 `json:"integration_id"`
	UserID        string                 `json:"user_id"`
	Changes       map[string]interface{} `json:"changes"`
	Timestamp     time.Time              `json:"timestamp"`
}

type IntegrationDeletedEvent struct {
	IntegrationID string    `json:"integration_id"`
	UserID        string    `json:"user_id"`
	Timestamp     time.Time `json:"timestamp"`
}

type IntegrationActivatedEvent struct {
	IntegrationID string    `json:"integration_id"`
	UserID        string    `json:"user_id"`
	Timestamp     time.Time `json:"timestamp"`
}

type IntegrationDeactivatedEvent struct {
	IntegrationID string    `json:"integration_id"`
	UserID        string    `json:"user_id"`
	Reason        string    `json:"reason,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

type IntegrationSyncedEvent struct {
	IntegrationID string                 `json:"integration_id"`
	ItemsSynced   int                    `json:"items_synced"`
	Errors        []string               `json:"errors,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

type IntegrationErrorEvent struct {
	IntegrationID string    `json:"integration_id"`
	Error         string    `json:"error"`
	ErrorCode     string    `json:"error_code,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// Storage Events
type FileUploadedEvent struct {
	FileID    string                 `json:"file_id"`
	UserID    string                 `json:"user_id"`
	FileName  string                 `json:"file_name"`
	Size      int64                  `json:"size"`
	MimeType  string                 `json:"mime_type"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type FileDeletedEvent struct {
	FileID    string    `json:"file_id"`
	UserID    string    `json:"user_id"`
	FileName  string    `json:"file_name"`
	Timestamp time.Time `json:"timestamp"`
}

type FileSharedEvent struct {
	FileID     string    `json:"file_id"`
	OwnerID    string    `json:"owner_id"`
	SharedWith []string  `json:"shared_with"`
	Public     bool      `json:"public"`
	Timestamp  time.Time `json:"timestamp"`
}

// Monitoring Events
type MetricRecordedEvent struct {
	Service   string                 `json:"service"`
	Metric    string                 `json:"metric"`
	Value     float64                `json:"value"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type AlertTriggeredEvent struct {
	AlertID   string    `json:"alert_id"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type AlertResolvedEvent struct {
	AlertID   string    `json:"alert_id"`
	Metric    string    `json:"metric"`
	Timestamp time.Time `json:"timestamp"`
}

type HealthCheckFailedEvent struct {
	Service      string    `json:"service"`
	Endpoint     string    `json:"endpoint"`
	Error        string    `json:"error"`
	ResponseTime int64     `json:"response_time_ms"`
	Timestamp    time.Time `json:"timestamp"`
}

// Config Events
type ConfigurationChangedEvent struct {
	ConfigKey string      `json:"config_key"`
	OldValue  interface{} `json:"old_value"`
	NewValue  interface{} `json:"new_value"`
	Scope     string      `json:"scope"`
	ScopeID   string      `json:"scope_id,omitempty"`
	ChangedBy string      `json:"changed_by"`
	Timestamp time.Time   `json:"timestamp"`
}

// Migration Events
type MigrationStartedEvent struct {
	Version   string    `json:"version"`
	Direction string    `json:"direction"` // up or down
	Timestamp time.Time `json:"timestamp"`
}

type MigrationCompletedEvent struct {
	Version   string    `json:"version"`
	Direction string    `json:"direction"`
	Duration  int64     `json:"duration_ms"`
	Timestamp time.Time `json:"timestamp"`
}

type MigrationFailedEvent struct {
	Version   string    `json:"version"`
	Direction string    `json:"direction"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

// Backup Events
type BackupStartedEvent struct {
	BackupID  string    `json:"backup_id"`
	Type      string    `json:"type"` // full, incremental, snapshot
	Timestamp time.Time `json:"timestamp"`
}

type BackupCompletedEvent struct {
	BackupID  string    `json:"backup_id"`
	Size      int64     `json:"size_bytes"`
	Duration  int64     `json:"duration_ms"`
	Timestamp time.Time `json:"timestamp"`
}

type BackupFailedEvent struct {
	BackupID  string    `json:"backup_id"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

type RestoreStartedEvent struct {
	BackupID  string    `json:"backup_id"`
	RestoreID string    `json:"restore_id"`
	Timestamp time.Time `json:"timestamp"`
}

type RestoreCompletedEvent struct {
	BackupID  string    `json:"backup_id"`
	RestoreID string    `json:"restore_id"`
	Duration  int64     `json:"duration_ms"`
	Timestamp time.Time `json:"timestamp"`
}

// Admin Events
type AdminActionEvent struct {
	AdminID   string                 `json:"admin_id"`
	Action    string                 `json:"action"`
	Target    string                 `json:"target"`
	TargetID  string                 `json:"target_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type SystemSettingsChangedEvent struct {
	AdminID   string                 `json:"admin_id"`
	Settings  map[string]interface{} `json:"settings"`
	Timestamp time.Time              `json:"timestamp"`
}

type ServiceMaintenanceEvent struct {
	Service   string    `json:"service"`
	Action    string    `json:"action"` // start, stop, restart, update
	AdminID   string    `json:"admin_id"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
