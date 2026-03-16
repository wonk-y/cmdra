package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when a job row does not exist for the requested key.
var ErrNotFound = errors.New("job not found")

// Job is the persisted metadata view for a command or file transfer.
type Job struct {
	ID                    string
	OwnerCN               string
	Kind                  string
	State                 string
	PID                   int64
	CommandArgv           []string
	CommandShell          string
	StartedAt             time.Time
	EndedAt               *time.Time
	ExitCode              *int32
	Signal                string
	OutputSizeBytes       int64
	TransferLocalPath     string
	TransferRemotePath    string
	TransferDirection     string
	TransferProgressBytes int64
	TransferTotalBytes    *int64
	ErrorMessage          string
}

// OutputRecord is one persisted output chunk from a job stream.
type OutputRecord struct {
	Sequence  int64
	Offset    int64
	Source    int
	Timestamp time.Time
	Data      []byte
}

// Store wraps the SQLite database used for metadata and output replay.
type Store struct {
	db *sql.DB
}

// Open opens the history database and creates the schema if needed.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;
`); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying SQLite connection.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) initSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  owner_cn TEXT NOT NULL,
  kind TEXT NOT NULL,
  state TEXT NOT NULL,
  pid INTEGER NOT NULL DEFAULT 0,
  command_argv_json TEXT NOT NULL DEFAULT '[]',
  command_shell TEXT NOT NULL DEFAULT '',
  started_at_unix_nano INTEGER NOT NULL,
  ended_at_unix_nano INTEGER,
  exit_code INTEGER,
  signal TEXT NOT NULL DEFAULT '',
  output_size_bytes INTEGER NOT NULL DEFAULT 0,
  next_sequence INTEGER NOT NULL DEFAULT 0,
  transfer_local_path TEXT NOT NULL DEFAULT '',
  transfer_remote_path TEXT NOT NULL DEFAULT '',
  transfer_direction TEXT NOT NULL DEFAULT '',
  transfer_progress_bytes INTEGER NOT NULL DEFAULT 0,
  transfer_total_bytes INTEGER,
  error_message TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_jobs_owner_started ON jobs(owner_cn, started_at_unix_nano DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_owner_state ON jobs(owner_cn, state);

CREATE TABLE IF NOT EXISTS job_output (
  job_id TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  offset_bytes INTEGER NOT NULL,
  source INTEGER NOT NULL,
  timestamp_unix_nano INTEGER NOT NULL,
  data BLOB NOT NULL,
  PRIMARY KEY (job_id, sequence),
  FOREIGN KEY(job_id) REFERENCES jobs(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_job_output_by_offset ON job_output(job_id, offset_bytes);
`)
	return err
}

// CreateJob inserts a new job row.
func (s *Store) CreateJob(job Job) error {
	argvJSON, err := json.Marshal(job.CommandArgv)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
INSERT INTO jobs (
  id, owner_cn, kind, state, pid, command_argv_json, command_shell,
  started_at_unix_nano, ended_at_unix_nano, exit_code, signal,
  output_size_bytes, next_sequence,
  transfer_local_path, transfer_remote_path, transfer_direction,
  transfer_progress_bytes, transfer_total_bytes, error_message
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?)
`,
		job.ID,
		job.OwnerCN,
		job.Kind,
		job.State,
		job.PID,
		string(argvJSON),
		job.CommandShell,
		job.StartedAt.UTC().UnixNano(),
		toNullInt64Time(job.EndedAt),
		toNullInt32(job.ExitCode),
		job.Signal,
		job.OutputSizeBytes,
		job.TransferLocalPath,
		job.TransferRemotePath,
		job.TransferDirection,
		job.TransferProgressBytes,
		toNullInt64(job.TransferTotalBytes),
		job.ErrorMessage,
	)
	return err
}

// UpdateJobProcess stores the runtime PID once a command successfully starts.
func (s *Store) UpdateJobProcess(id string, pid int64) error {
	result, err := s.db.Exec(`UPDATE jobs SET pid = ? WHERE id = ?`, pid, id)
	if err != nil {
		return err
	}
	return mustAffectOne(result)
}

// CompleteJob marks a job as finished and stores its final metadata.
func (s *Store) CompleteJob(id, state string, endedAt time.Time, exitCode *int32, signal, errorMessage string) error {
	result, err := s.db.Exec(`
UPDATE jobs
SET state = ?, ended_at_unix_nano = ?, exit_code = ?, signal = ?, error_message = ?
WHERE id = ?`,
		state,
		endedAt.UTC().UnixNano(),
		toNullInt32(exitCode),
		signal,
		errorMessage,
		id,
	)
	if err != nil {
		return err
	}
	return mustAffectOne(result)
}

// UpdateTransferProgress updates the running byte counters for an upload or download.
func (s *Store) UpdateTransferProgress(id string, progress int64, total *int64) error {
	result, err := s.db.Exec(`
UPDATE jobs
SET transfer_progress_bytes = ?, transfer_total_bytes = COALESCE(?, transfer_total_bytes)
WHERE id = ?`,
		progress,
		toNullInt64(total),
		id,
	)
	if err != nil {
		return err
	}
	return mustAffectOne(result)
}

// AppendOutput appends one output chunk and advances the replay sequence atomically.
func (s *Store) AppendOutput(jobID string, source int, ts time.Time, data []byte) (int64, int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var outputSize int64
	var nextSeq int64
	if err := tx.QueryRow(`SELECT output_size_bytes, next_sequence FROM jobs WHERE id = ?`, jobID).Scan(&outputSize, &nextSeq); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, ErrNotFound
		}
		return 0, 0, err
	}

	seq := nextSeq + 1
	offset := outputSize
	if _, err := tx.Exec(`
INSERT INTO job_output (job_id, sequence, offset_bytes, source, timestamp_unix_nano, data)
VALUES (?, ?, ?, ?, ?, ?)`,
		jobID, seq, offset, source, ts.UTC().UnixNano(), data,
	); err != nil {
		return 0, 0, err
	}
	if _, err := tx.Exec(`
UPDATE jobs
SET output_size_bytes = output_size_bytes + ?, next_sequence = ?
WHERE id = ?`,
		len(data), seq, jobID,
	); err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return offset, seq, nil
}

// StreamOutput replays persisted output records starting at the requested byte offset.
func (s *Store) StreamOutput(ctx context.Context, jobID string, offset int64, fn func(OutputRecord) error) error {
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT sequence, offset_bytes, source, timestamp_unix_nano, data
FROM job_output
WHERE job_id = ? AND offset_bytes >= ?
ORDER BY offset_bytes ASC
`, jobID, offset)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var rec OutputRecord
		var nanos int64
		if err := rows.Scan(&rec.Sequence, &rec.Offset, &rec.Source, &nanos, &rec.Data); err != nil {
			return err
		}
		rec.Timestamp = time.Unix(0, nanos).UTC()
		if err := fn(rec); err != nil {
			return err
		}
	}
	return rows.Err()
}

// DeleteOutput removes persisted output rows once a caller no longer needs replay.
func (s *Store) DeleteOutput(jobID string) error {
	_, err := s.db.Exec(`DELETE FROM job_output WHERE job_id = ?`, jobID)
	return err
}

// DeleteJob removes one persisted job row and its output by ID.
func (s *Store) DeleteJob(jobID string) error {
	result, err := s.db.Exec(`DELETE FROM jobs WHERE id = ?`, jobID)
	if err != nil {
		return err
	}
	return mustAffectOne(result)
}

// GetJob returns a job owned by the given CN.
func (s *Store) GetJob(jobID, ownerCN string) (Job, error) {
	row := s.db.QueryRow(`
SELECT
  id, owner_cn, kind, state, pid, command_argv_json, command_shell,
  started_at_unix_nano, ended_at_unix_nano, exit_code, signal,
  output_size_bytes,
  transfer_local_path, transfer_remote_path, transfer_direction,
  transfer_progress_bytes, transfer_total_bytes, error_message
FROM jobs
WHERE id = ? AND owner_cn = ?
`, jobID, ownerCN)
	return scanJob(row)
}

// GetJobByID returns a job regardless of owner so callers can distinguish
// authorization failures from a truly missing ID.
func (s *Store) GetJobByID(jobID string) (Job, error) {
	row := s.db.QueryRow(`
SELECT
  id, owner_cn, kind, state, pid, command_argv_json, command_shell,
  started_at_unix_nano, ended_at_unix_nano, exit_code, signal,
  output_size_bytes,
  transfer_local_path, transfer_remote_path, transfer_direction,
  transfer_progress_bytes, transfer_total_bytes, error_message
FROM jobs
WHERE id = ?
`, jobID)
	return scanJob(row)
}

// ListJobs returns jobs for one CN, optionally restricted to currently running jobs.
func (s *Store) ListJobs(ownerCN string, runningOnly bool) ([]Job, error) {
	query := `
SELECT
  id, owner_cn, kind, state, pid, command_argv_json, command_shell,
  started_at_unix_nano, ended_at_unix_nano, exit_code, signal,
  output_size_bytes,
  transfer_local_path, transfer_remote_path, transfer_direction,
  transfer_progress_bytes, transfer_total_bytes, error_message
FROM jobs
WHERE owner_cn = ?
`
	args := []any{ownerCN}
	if runningOnly {
		query += ` AND state = 'running'`
	}
	query += ` ORDER BY started_at_unix_nano DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Job, 0)
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanJob(s scanner) (Job, error) {
	var job Job
	var argvJSON string
	var startedAtNanos int64
	var endedAtNanos sql.NullInt64
	var exitCode sql.NullInt64
	var totalBytes sql.NullInt64
	if err := s.Scan(
		&job.ID, &job.OwnerCN, &job.Kind, &job.State, &job.PID, &argvJSON, &job.CommandShell,
		&startedAtNanos, &endedAtNanos, &exitCode, &job.Signal, &job.OutputSizeBytes,
		&job.TransferLocalPath, &job.TransferRemotePath, &job.TransferDirection,
		&job.TransferProgressBytes, &totalBytes, &job.ErrorMessage,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Job{}, ErrNotFound
		}
		return Job{}, err
	}
	job.StartedAt = time.Unix(0, startedAtNanos).UTC()
	if endedAtNanos.Valid {
		t := time.Unix(0, endedAtNanos.Int64).UTC()
		job.EndedAt = &t
	}
	if exitCode.Valid {
		c := int32(exitCode.Int64)
		job.ExitCode = &c
	}
	if totalBytes.Valid {
		n := totalBytes.Int64
		job.TransferTotalBytes = &n
	}
	if err := json.Unmarshal([]byte(argvJSON), &job.CommandArgv); err != nil {
		return Job{}, fmt.Errorf("decode command argv for job %s: %w", job.ID, err)
	}
	return job, nil
}

func toNullInt32(v *int32) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Valid: true, Int64: int64(*v)}
}

func toNullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Valid: true, Int64: *v}
}

func toNullInt64Time(v *time.Time) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Valid: true, Int64: v.UTC().UnixNano()}
}

func mustAffectOne(result sql.Result) error {
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
