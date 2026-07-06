package backup

import (
	"errors"
	"time"
)

var (
	ErrS3NotConfigured = errors.New("backup S3 storage is not configured")
	ErrNotFound        = errors.New("backup record not found")
	ErrInProgress      = errors.New("a backup is already in progress")
	ErrRestoreRunning  = errors.New("a restore is already in progress")
	ErrNotCompleted    = errors.New("backup is not completed")
	ErrInvalidType     = errors.New("invalid backup type")
)

const (
	TypeConfig = "config"
	TypeFull   = "full"
)

const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// S3Config holds S3-compatible object storage credentials.
type S3Config struct {
	Endpoint           string `json:"endpoint"`
	Region             string `json:"region"`
	Bucket             string `json:"bucket"`
	AccessKeyID        string `json:"access_key_id"`
	SecretAccessKey    string `json:"secret_access_key,omitempty"`
	SecretConfigured   bool   `json:"secret_configured"`
	Prefix             string `json:"prefix"`
	ForcePathStyle     bool   `json:"force_path_style"`
}

func (c *S3Config) IsConfigured() bool {
	return c.Bucket != "" && c.AccessKeyID != "" && c.SecretAccessKey != ""
}

// ScheduleConfig is the user-friendly schedule (no raw cron in API).
type ScheduleConfig struct {
	Enabled       bool `json:"enabled"`
	StartHour     int  `json:"start_hour"`     // 0-23
	StartMinute   int  `json:"start_minute"`   // 0-59
	BackupsPerDay int  `json:"backups_per_day"` // 1-24
	BackupType    string `json:"backup_type"` // config | full
	RetainDays    int  `json:"retain_days"`   // 0 = unlimited
	RetainCount   int  `json:"retain_count"`  // 0 = unlimited
}

// Record is one backup job stored in settings JSON.
type Record struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	BackupType    string `json:"backup_type"`
	FileName      string `json:"file_name"`
	S3Key         string `json:"s3_key"`
	SizeBytes     int64  `json:"size_bytes"`
	TriggeredBy   string `json:"triggered_by"`
	ErrorMessage  string `json:"error_message,omitempty"`
	StartedAt     string `json:"started_at"`
	FinishedAt    string `json:"finished_at,omitempty"`
	Progress      string `json:"progress,omitempty"`
	RestoreStatus string `json:"restore_status,omitempty"`
	RestoreError  string `json:"restore_error,omitempty"`
	RestoredAt    string `json:"restored_at,omitempty"`
}

// CreateRequest is the body for POST /admin/backups.
type CreateRequest struct {
	BackupType string `json:"backup_type"`
}

// RestoreRequest is the body for POST /admin/backups/:id/restore.
type RestoreRequest struct {
	Password  string `json:"password"`
	PreBackup *bool  `json:"pre_backup"`
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}