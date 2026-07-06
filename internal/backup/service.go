package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"golang.org/x/crypto/bcrypt"

	"github.com/voidmind-io/voidllm/internal/config"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/pkg/crypto"
)

// Service orchestrates cloud backups and restores.
type Service struct {
	db      *db.DB
	dbCfg   config.DatabaseConfig
	encKey  []byte
	log     *slog.Logger
	dumper  *FullDumper
	onRestore func(context.Context) error

	opMu      sync.Mutex
	backingUp bool
	restoring bool

	storeMu sync.Mutex
	store   ObjectStore
	s3Cfg   *S3Config

	recordsMu sync.Mutex

	cronMu      sync.Mutex
	cronSched   *cron.Cron
	cronEntryIDs []cron.EntryID

	wg           sync.WaitGroup
	shuttingDown atomic.Bool
	bgCtx        context.Context
	bgCancel     context.CancelFunc
}

func NewService(database *db.DB, cfg config.DatabaseConfig, encKey []byte, log *slog.Logger, onRestore func(context.Context) error) *Service {
	bgCtx, bgCancel := context.WithCancel(context.Background())
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		db:      database,
		dbCfg:   cfg,
		encKey:  encKey,
		log:     log.With(slog.String("component", "backup")),
		dumper:  NewFullDumper(cfg),
		onRestore: onRestore,
		bgCtx:   bgCtx,
		bgCancel: bgCancel,
	}
}

func (s *Service) Start() {
	s.cronSched = cron.New()
	s.cronSched.Start()
	s.recoverStaleRecords()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	schedule, err := s.GetSchedule(ctx)
	if err == nil && schedule.Enabled {
		_ = s.applySchedule(schedule)
	}
}

func (s *Service) Stop() {
	s.shuttingDown.Store(true)
	s.bgCancel()
	if s.cronSched != nil {
		ctx := s.cronSched.Stop()
		<-ctx.Done()
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Minute):
	}
}

func (s *Service) recoverStaleRecords() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	records, err := s.loadRecords(ctx)
	if err != nil {
		return
	}
	for i := range records {
		changed := false
		if records[i].Status == StatusRunning {
			records[i].Status = StatusFailed
			records[i].ErrorMessage = "interrupted by server restart"
			records[i].Progress = ""
			records[i].FinishedAt = nowRFC3339()
			changed = true
		}
		if records[i].RestoreStatus == StatusRunning {
			records[i].RestoreStatus = StatusFailed
			records[i].RestoreError = "interrupted by server restart"
			changed = true
		}
		if changed {
			_ = s.saveRecord(ctx, &records[i])
		}
	}
}

func (s *Service) GetS3Config(ctx context.Context) (*S3Config, error) {
	raw, err := s.db.GetSetting(ctx, KeyS3Config)
	if err != nil || raw == "" {
		return &S3Config{Prefix: "backups"}, nil
	}
	var cfg S3Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, err
	}
	if cfg.SecretAccessKey != "" {
		plain, err := crypto.DecryptString(cfg.SecretAccessKey, s.encKey, []byte("backup-s3-secret"))
		if err == nil {
			cfg.SecretAccessKey = plain
		}
	}
	out := cfg
	out.SecretConfigured = cfg.SecretAccessKey != ""
	out.SecretAccessKey = ""
	return &out, nil
}

func (s *Service) UpdateS3Config(ctx context.Context, input S3Config) (*S3Config, error) {
	current, _ := s.GetS3Config(ctx)
	secret := input.SecretAccessKey
	if secret == "" && current != nil {
		full, err := s.loadS3Secret(ctx)
		if err != nil {
			return nil, err
		}
		secret = full
	}
	encSecret, err := crypto.EncryptString(secret, s.encKey, []byte("backup-s3-secret"))
	if err != nil {
		return nil, err
	}
	stored := S3Config{
		Endpoint:        input.Endpoint,
		Region:          input.Region,
		Bucket:          input.Bucket,
		AccessKeyID:     input.AccessKeyID,
		SecretAccessKey: encSecret,
		Prefix:          input.Prefix,
		ForcePathStyle:  input.ForcePathStyle,
	}
	if stored.Prefix == "" {
		stored.Prefix = "backups"
	}
	b, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}
	if err := s.db.SetSetting(ctx, KeyS3Config, string(b)); err != nil {
		return nil, err
	}
	s.storeMu.Lock()
	s.store = nil
	s.s3Cfg = nil
	s.storeMu.Unlock()
	return s.GetS3Config(ctx)
}

func (s *Service) loadS3Secret(ctx context.Context) (string, error) {
	raw, err := s.db.GetSetting(ctx, KeyS3Config)
	if err != nil || raw == "" {
		return "", nil
	}
	var cfg S3Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return "", err
	}
	if cfg.SecretAccessKey == "" {
		return "", nil
	}
	return crypto.DecryptString(cfg.SecretAccessKey, s.encKey, []byte("backup-s3-secret"))
}

func (s *Service) TestS3Connection(ctx context.Context, input S3Config) error {
	secret := input.SecretAccessKey
	if secret == "" {
		var err error
		secret, err = s.loadS3Secret(ctx)
		if err != nil {
			return err
		}
	}
	cfg := input
	cfg.SecretAccessKey = secret
	if !cfg.IsConfigured() {
		return ErrS3NotConfigured
	}
	store, err := NewObjectStore(ctx, &cfg)
	if err != nil {
		return err
	}
	return store.HeadBucket(ctx)
}

func (s *Service) GetSchedule(ctx context.Context) (*ScheduleConfig, error) {
	raw, err := s.db.GetSetting(ctx, KeySchedule)
	if err != nil || raw == "" {
		return defaultSchedule(), nil
	}
	var cfg ScheduleConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, err
	}
	if cfg.BackupType == "" {
		cfg.BackupType = TypeConfig
	}
	return &cfg, nil
}

func defaultSchedule() *ScheduleConfig {
	return &ScheduleConfig{
		BackupType:    TypeConfig,
		BackupsPerDay: 1,
		RetainDays:    14,
		RetainCount:   30,
	}
}

func (s *Service) UpdateSchedule(ctx context.Context, input ScheduleConfig) (*ScheduleConfig, error) {
	if input.BackupsPerDay < 1 || input.BackupsPerDay > 24 {
		return nil, fmt.Errorf("backups_per_day must be between 1 and 24")
	}
	if input.BackupType != TypeConfig && input.BackupType != TypeFull {
		return nil, ErrInvalidType
	}
	if _, err := SlotTimes(input.StartHour, input.StartMinute, input.BackupsPerDay); err != nil {
		return nil, err
	}
	b, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	if err := s.db.SetSetting(ctx, KeySchedule, string(b)); err != nil {
		return nil, err
	}
	if err := s.applySchedule(&input); err != nil {
		return nil, err
	}
	return &input, nil
}

func (s *Service) SlotPreview(schedule *ScheduleConfig) ([]string, error) {
	return SlotTimes(schedule.StartHour, schedule.StartMinute, schedule.BackupsPerDay)
}

func (s *Service) applySchedule(schedule *ScheduleConfig) error {
	s.cronMu.Lock()
	defer s.cronMu.Unlock()
	for _, id := range s.cronEntryIDs {
		s.cronSched.Remove(id)
	}
	s.cronEntryIDs = nil
	if !schedule.Enabled {
		return nil
	}
	slots, err := SlotTimes(schedule.StartHour, schedule.StartMinute, schedule.BackupsPerDay)
	if err != nil {
		return err
	}
	exprs, err := CronExprs(slots)
	if err != nil {
		return err
	}
	backupType := schedule.BackupType
	for _, expr := range exprs {
		id, err := s.cronSched.AddFunc(expr, func() {
			if s.shuttingDown.Load() {
				return
			}
			ctx, cancel := context.WithTimeout(s.bgCtx, 30*time.Minute)
			defer cancel()
			if _, err := s.runBackup(ctx, backupType, "scheduled"); err != nil {
				s.log.Error("scheduled backup failed", slog.String("error", err.Error()))
			}
			_ = s.cleanupOldBackups(ctx)
		})
		if err != nil {
			return err
		}
		s.cronEntryIDs = append(s.cronEntryIDs, id)
	}
	return nil
}

func (s *Service) ListBackups(ctx context.Context) ([]Record, error) {
	return s.loadRecords(ctx)
}

func (s *Service) GetBackup(ctx context.Context, id string) (*Record, error) {
	records, err := s.loadRecords(ctx)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].ID == id {
			return &records[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) StartBackup(ctx context.Context, backupType string) (*Record, error) {
	if backupType != TypeConfig && backupType != TypeFull {
		return nil, ErrInvalidType
	}
	return s.runBackup(ctx, backupType, "manual")
}

func (s *Service) runBackupSync(ctx context.Context, backupType, triggeredBy string) (*Record, error) {
	if s.shuttingDown.Load() {
		return nil, fmt.Errorf("server is shutting down")
	}
	s.opMu.Lock()
	if s.backingUp {
		s.opMu.Unlock()
		return nil, ErrInProgress
	}
	s.backingUp = true
	s.opMu.Unlock()
	defer func() {
		s.opMu.Lock()
		s.backingUp = false
		s.opMu.Unlock()
	}()

	store, s3cfg, err := s.getStore(ctx)
	if err != nil {
		return nil, err
	}
	rec := &Record{
		ID:          uuid.NewString(),
		Status:      StatusRunning,
		BackupType:  backupType,
		TriggeredBy: triggeredBy,
		StartedAt:   nowRFC3339(),
		Progress:    "dumping",
	}
	fileName := backupFileName(backupType, s.dbCfg.Driver)
	key := s3Key(s3cfg.Prefix, fileName)
	rec.FileName = fileName
	rec.S3Key = key
	if err := s.saveRecord(ctx, rec); err != nil {
		return nil, err
	}
	if err := s.executeBackup(ctx, store, rec); err != nil {
		rec.Status = StatusFailed
		rec.ErrorMessage = err.Error()
		rec.FinishedAt = nowRFC3339()
		_ = s.saveRecord(ctx, rec)
		return nil, err
	}
	rec.Status = StatusCompleted
	rec.FinishedAt = nowRFC3339()
	_ = s.saveRecord(ctx, rec)
	return rec, nil
}

func (s *Service) runBackup(ctx context.Context, backupType, triggeredBy string) (*Record, error) {
	if s.shuttingDown.Load() {
		return nil, fmt.Errorf("server is shutting down")
	}
	s.opMu.Lock()
	if s.backingUp {
		s.opMu.Unlock()
		return nil, ErrInProgress
	}
	s.backingUp = true
	s.opMu.Unlock()
	defer func() {
		s.opMu.Lock()
		s.backingUp = false
		s.opMu.Unlock()
	}()

	store, s3cfg, err := s.getStore(ctx)
	if err != nil {
		return nil, err
	}

	rec := &Record{
		ID:          uuid.NewString(),
		Status:      StatusRunning,
		BackupType:  backupType,
		TriggeredBy: triggeredBy,
		StartedAt:   nowRFC3339(),
		Progress:    "dumping",
	}
	fileName := backupFileName(backupType, s.dbCfg.Driver)
	key := s3Key(s3cfg.Prefix, fileName)
	rec.FileName = fileName
	rec.S3Key = key
	if err := s.saveRecord(ctx, rec); err != nil {
		return nil, err
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		bgCtx, cancel := context.WithTimeout(s.bgCtx, 30*time.Minute)
		defer cancel()
		if err := s.executeBackup(bgCtx, store, rec); err != nil {
			rec.Status = StatusFailed
			rec.ErrorMessage = err.Error()
			rec.Progress = ""
			rec.FinishedAt = nowRFC3339()
			_ = s.saveRecord(bgCtx, rec)
			return
		}
		rec.Status = StatusCompleted
		rec.Progress = ""
		rec.FinishedAt = nowRFC3339()
		_ = s.saveRecord(bgCtx, rec)
		if triggeredBy == "scheduled" {
			_ = s.cleanupOldBackups(bgCtx)
		}
	}()

	return rec, nil
}

func (s *Service) executeBackup(ctx context.Context, store ObjectStore, rec *Record) error {
	var reader io.ReadCloser
	var err error
	var contentType string

	if rec.BackupType == TypeConfig {
		payload, err := dumpConfigTables(ctx, s.db.SQL(), s.dbCfg.Driver)
		if err != nil {
			return err
		}
		raw, err := encodeConfigPayload(payload)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(raw); err != nil {
			gz.Close()
			return err
		}
		gz.Close()
		reader = io.NopCloser(bytes.NewReader(buf.Bytes()))
		contentType = "application/gzip"
	} else {
		reader, err = s.dumper.Dump(ctx)
		if err != nil {
			return err
		}
		contentType = "application/gzip"
	}
	defer reader.Close()

	rec.Progress = "uploading"
	_ = s.saveRecord(ctx, rec)

	size, err := store.Upload(ctx, rec.S3Key, reader, contentType)
	if err != nil {
		return err
	}
	rec.SizeBytes = size

	headSize, err := store.HeadObject(ctx, rec.S3Key)
	if err != nil {
		return fmt.Errorf("verify upload: %w", err)
	}
	if headSize != size {
		return fmt.Errorf("upload size mismatch: sent %d, head %d", size, headSize)
	}
	return nil
}

func (s *Service) DeleteBackup(ctx context.Context, id string) error {
	rec, err := s.GetBackup(ctx, id)
	if err != nil {
		return err
	}
	store, _, err := s.getStore(ctx)
	if err != nil {
		return err
	}
	if rec.S3Key != "" {
		_ = store.Delete(ctx, rec.S3Key)
	}
	return s.removeRecord(ctx, id)
}

func (s *Service) DownloadURL(ctx context.Context, id string) (string, error) {
	rec, err := s.GetBackup(ctx, id)
	if err != nil {
		return "", err
	}
	if rec.Status != StatusCompleted {
		return "", ErrNotCompleted
	}
	store, _, err := s.getStore(ctx)
	if err != nil {
		return "", err
	}
	return store.PresignURL(ctx, rec.S3Key, time.Hour)
}

func (s *Service) StartRestore(ctx context.Context, id, userID, password string, preBackup bool) error {
	if s.shuttingDown.Load() {
		return fmt.Errorf("server is shutting down")
	}
	_, hash, err := s.db.GetUserPasswordHashByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("verify password: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("invalid password")
	}

	s.opMu.Lock()
	if s.restoring || s.backingUp {
		s.opMu.Unlock()
		if s.restoring {
			return ErrRestoreRunning
		}
		return ErrInProgress
	}
	s.restoring = true
	s.opMu.Unlock()

	rec, err := s.GetBackup(ctx, id)
	if err != nil {
		s.opMu.Lock()
		s.restoring = false
		s.opMu.Unlock()
		return err
	}
	if rec.Status != StatusCompleted {
		s.opMu.Lock()
		s.restoring = false
		s.opMu.Unlock()
		return ErrNotCompleted
	}

	rec.RestoreStatus = StatusRunning
	rec.RestoreError = ""
	_ = s.saveRecord(ctx, rec)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			s.opMu.Lock()
			s.restoring = false
			s.opMu.Unlock()
		}()
		bgCtx, cancel := context.WithTimeout(s.bgCtx, 30*time.Minute)
		defer cancel()
		if preBackup {
			preRec, err := s.runBackupSync(bgCtx, rec.BackupType, "pre_restore")
			if err != nil {
				rec.RestoreStatus = StatusFailed
				rec.RestoreError = "pre-backup failed: " + err.Error()
				_ = s.saveRecord(bgCtx, rec)
				return
			}
			s.log.Info("pre-restore backup completed", slog.String("id", preRec.ID))
		}
		if err := s.executeRestore(bgCtx, rec); err != nil {
			rec.RestoreStatus = StatusFailed
			rec.RestoreError = err.Error()
			_ = s.saveRecord(bgCtx, rec)
			return
		}
		rec.RestoreStatus = StatusCompleted
		rec.RestoredAt = nowRFC3339()
		_ = s.saveRecord(bgCtx, rec)
		if s.onRestore != nil {
			_ = s.onRestore(bgCtx)
		}
	}()
	return nil
}

func (s *Service) executeRestore(ctx context.Context, rec *Record) error {
	store, _, err := s.getStore(ctx)
	if err != nil {
		return err
	}
	body, err := store.Download(ctx, rec.S3Key)
	if err != nil {
		return err
	}
	defer body.Close()

	switch rec.BackupType {
	case TypeConfig:
		gz, err := gzip.NewReader(body)
		if err != nil {
			return err
		}
		defer gz.Close()
		data, err := io.ReadAll(gz)
		if err != nil {
			return err
		}
		payload, err := decodeConfigPayload(data)
		if err != nil {
			return err
		}
		return restoreConfigTables(ctx, s.db.SQL(), s.db.Dialect(), payload)
	case TypeFull:
		return s.dumper.Restore(ctx, body)
	default:
		return ErrInvalidType
	}
}

func (s *Service) cleanupOldBackups(ctx context.Context) error {
	schedule, err := s.GetSchedule(ctx)
	if err != nil {
		return err
	}
	if schedule.RetainDays <= 0 && schedule.RetainCount <= 0 {
		return nil
	}
	records, err := s.loadRecords(ctx)
	if err != nil {
		return err
	}
	var completed []Record
	for _, r := range records {
		if r.Status == StatusCompleted {
			completed = append(completed, r)
		}
	}
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].StartedAt > completed[j].StartedAt
	})
	cutoff := time.Now().UTC().AddDate(0, 0, -schedule.RetainDays)
	for i, r := range completed {
		tooMany := schedule.RetainCount > 0 && i >= schedule.RetainCount
		tooOld := false
		if schedule.RetainDays > 0 {
			if t, err := time.Parse(time.RFC3339, r.StartedAt); err == nil && t.Before(cutoff) {
				tooOld = true
			}
		}
		if tooMany || tooOld {
			_ = s.DeleteBackup(ctx, r.ID)
		}
	}
	return nil
}

func (s *Service) getStore(ctx context.Context) (ObjectStore, *S3Config, error) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	if s.store != nil && s.s3Cfg != nil {
		return s.store, s.s3Cfg, nil
	}
	raw, err := s.db.GetSetting(ctx, KeyS3Config)
	if err != nil || raw == "" {
		return nil, nil, ErrS3NotConfigured
	}
	var cfg S3Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, nil, err
	}
	secret, err := crypto.DecryptString(cfg.SecretAccessKey, s.encKey, []byte("backup-s3-secret"))
	if err != nil {
		return nil, nil, err
	}
	cfg.SecretAccessKey = secret
	if !cfg.IsConfigured() {
		return nil, nil, ErrS3NotConfigured
	}
	store, err := NewObjectStore(ctx, &cfg)
	if err != nil {
		return nil, nil, err
	}
	s.store = store
	s.s3Cfg = &cfg
	return store, &cfg, nil
}

func (s *Service) loadRecords(ctx context.Context) ([]Record, error) {
	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()
	raw, err := s.db.GetSetting(ctx, KeyRecords)
	if err != nil || raw == "" {
		return []Record{}, nil
	}
	var records []Record
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *Service) saveRecord(ctx context.Context, rec *Record) error {
	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()
	records, err := s.loadRecordsUnlocked(ctx)
	if err != nil {
		return err
	}
	found := false
	for i := range records {
		if records[i].ID == rec.ID {
			records[i] = *rec
			found = true
			break
		}
	}
	if !found {
		records = append([]Record{*rec}, records...)
	}
	if len(records) > maxBackupRecords {
		records = records[:maxBackupRecords]
	}
	b, err := json.Marshal(records)
	if err != nil {
		return err
	}
	return s.db.SetSetting(ctx, KeyRecords, string(b))
}

func (s *Service) loadRecordsUnlocked(ctx context.Context) ([]Record, error) {
	raw, err := s.db.GetSetting(ctx, KeyRecords)
	if err != nil || raw == "" {
		return []Record{}, nil
	}
	var records []Record
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *Service) removeRecord(ctx context.Context, id string) error {
	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()
	records, err := s.loadRecordsUnlocked(ctx)
	if err != nil {
		return err
	}
	out := make([]Record, 0, len(records))
	for _, r := range records {
		if r.ID != id {
			out = append(out, r)
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return s.db.SetSetting(ctx, KeyRecords, string(b))
}