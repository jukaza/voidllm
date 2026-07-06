package backup

const (
	KeyS3Config = "backup.s3"
	KeySchedule = "backup.schedule"
	KeyRecords  = "backup.records"
)

const maxBackupRecords = 100