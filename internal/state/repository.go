package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Saad7890-web/orbit/internal/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type StackRecord struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Version     string            `json:"version,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	ConfigHash  string            `json:"configHash,omitempty"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type ServiceRecord struct {
	StackName     string               `json:"stackName"`
	Name          string               `json:"name"`
	ConfigHash    string               `json:"configHash,omitempty"`
	Status        models.LifecycleStatus `json:"status,omitempty"`
	HealthStatus  models.HealthStatus   `json:"healthStatus,omitempty"`
	LastError     string               `json:"lastError,omitempty"`
	LastSeenAt    *time.Time           `json:"lastSeenAt,omitempty"`
	Service       models.Service       `json:"service"`
	UpdatedAt     time.Time            `json:"updatedAt"`
}

type JobRecord struct {
	StackName    string            `json:"stackName"`
	Name         string            `json:"name"`
	ConfigHash   string            `json:"configHash,omitempty"`
	LastStatus   models.ExecutionStatus `json:"lastStatus,omitempty"`
	LastRunAt    *time.Time        `json:"lastRunAt,omitempty"`
	LastError    string            `json:"lastError,omitempty"`
	Job          models.Job        `json:"job"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

type TriggerRecord struct {
	StackName    string            `json:"stackName"`
	Name         string            `json:"name"`
	ConfigHash   string            `json:"configHash,omitempty"`
	LastStatus   models.ExecutionStatus `json:"lastStatus,omitempty"`
	LastFiredAt  *time.Time        `json:"lastFiredAt,omitempty"`
	LastError    string            `json:"lastError,omitempty"`
	Trigger      models.Trigger    `json:"trigger"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

func (r *Repository) SaveStack(ctx context.Context, stack models.Stack, configHash string) error {
	if err := stack.Validate(); err != nil {
		return err
	}
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}

	labelsJSON, err := json.Marshal(stack.Labels)
	if err != nil {
		return fmt.Errorf("marshal stack labels: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO stacks(name, description, version, labels_json, config_hash, updated_at, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			description=excluded.description,
			version=excluded.version,
			labels_json=excluded.labels_json,
			config_hash=excluded.config_hash,
			updated_at=excluded.updated_at
	`, stack.Name, stack.Description, stack.Version, string(labelsJSON), configHash, now, now)
	if err != nil {
		return fmt.Errorf("save stack %q: %w", stack.Name, err)
	}

	return nil
}

func (r *Repository) GetStack(ctx context.Context, name string) (*StackRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT name, description, version, labels_json, config_hash, updated_at
		FROM stacks
		WHERE name = ?
	`, name)

	var rec StackRecord
	var labelsJSON string
	var updatedAt string

	if err := row.Scan(&rec.Name, &rec.Description, &rec.Version, &labelsJSON, &rec.ConfigHash, &updatedAt); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(labelsJSON), &rec.Labels); err != nil {
		return nil, fmt.Errorf("unmarshal labels for stack %q: %w", name, err)
	}

	t, err := time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse updatedAt for stack %q: %w", name, err)
	}
	rec.UpdatedAt = t

	return &rec, nil
}

func (r *Repository) ListStacks(ctx context.Context) ([]StackRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT name, description, version, labels_json, config_hash, updated_at
		FROM stacks
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StackRecord
	for rows.Next() {
		var rec StackRecord
		var labelsJSON string
		var updatedAt string
		if err := rows.Scan(&rec.Name, &rec.Description, &rec.Version, &labelsJSON, &rec.ConfigHash, &updatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(labelsJSON), &rec.Labels); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, err
		}
		rec.UpdatedAt = t
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertService(ctx context.Context, stackName string, svc models.Service, configHash string, status models.LifecycleStatus, health models.HealthStatus, lastError string) error {
	if err := svc.Validate(); err != nil {
		return err
	}
	svcJSON, err := json.Marshal(svc)
	if err != nil {
		return fmt.Errorf("marshal service: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO services(
			stack_name, name, spec_json, config_hash,
			status, health_status, last_error, updated_at, created_at
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(stack_name, name) DO UPDATE SET
			spec_json=excluded.spec_json,
			config_hash=excluded.config_hash,
			status=excluded.status,
			health_status=excluded.health_status,
			last_error=excluded.last_error,
			updated_at=excluded.updated_at
	`, stackName, svc.Name, string(svcJSON), configHash, status, health, lastError, now, now)
	if err != nil {
		return fmt.Errorf("upsert service %q: %w", svc.Name, err)
	}
	return nil
}

func (r *Repository) ListServices(ctx context.Context, stackName string) ([]ServiceRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT stack_name, name, spec_json, config_hash, status, health_status, last_error, last_seen_at, updated_at
		FROM services
		WHERE stack_name = ?
		ORDER BY name
	`, stackName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ServiceRecord
	for rows.Next() {
		var rec ServiceRecord
		var svcJSON string
		var lastSeenAt sql.NullString
		var updatedAt string

		if err := rows.Scan(
			&rec.StackName, &rec.Name, &svcJSON, &rec.ConfigHash,
			&rec.Status, &rec.HealthStatus, &rec.LastError, &lastSeenAt, &updatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(svcJSON), &rec.Service); err != nil {
			return nil, err
		}

		if lastSeenAt.Valid && lastSeenAt.String != "" {
			t, err := time.Parse(time.RFC3339Nano, lastSeenAt.String)
			if err != nil {
				return nil, err
			}
			rec.LastSeenAt = &t
		}

		t, err := time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, err
		}
		rec.UpdatedAt = t

		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertJob(ctx context.Context, stackName string, job models.Job, configHash string, lastStatus models.ExecutionStatus, lastError string, lastRunAt *time.Time) error {
	if err := job.Validate(); err != nil {
		return err
	}
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	var lastRunAtStr *string
	if lastRunAt != nil {
		s := lastRunAt.UTC().Format(time.RFC3339Nano)
		lastRunAtStr = &s
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO jobs(
			stack_name, name, spec_json, config_hash,
			last_status, last_error, last_run_at, updated_at, created_at
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(stack_name, name) DO UPDATE SET
			spec_json=excluded.spec_json,
			config_hash=excluded.config_hash,
			last_status=excluded.last_status,
			last_error=excluded.last_error,
			last_run_at=excluded.last_run_at,
			updated_at=excluded.updated_at
	`, stackName, job.Name, string(jobJSON), configHash, lastStatus, lastError, lastRunAtStr, now, now)
	if err != nil {
		return fmt.Errorf("upsert job %q: %w", job.Name, err)
	}
	return nil
}

func (r *Repository) ListJobs(ctx context.Context, stackName string) ([]JobRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT stack_name, name, spec_json, config_hash, last_status, last_error, last_run_at, updated_at
		FROM jobs
		WHERE stack_name = ?
		ORDER BY name
	`, stackName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []JobRecord
	for rows.Next() {
		var rec JobRecord
		var jobJSON string
		var lastRunAt sql.NullString
		var updatedAt string

		if err := rows.Scan(
			&rec.StackName, &rec.Name, &jobJSON, &rec.ConfigHash,
			&rec.LastStatus, &rec.LastError, &lastRunAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(jobJSON), &rec.Job); err != nil {
			return nil, err
		}
		if lastRunAt.Valid && lastRunAt.String != "" {
			t, err := time.Parse(time.RFC3339Nano, lastRunAt.String)
			if err != nil {
				return nil, err
			}
			rec.LastRunAt = &t
		}
		t, err := time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, err
		}
		rec.UpdatedAt = t

		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertTrigger(ctx context.Context, stackName string, trigger models.Trigger, configHash string, lastStatus models.ExecutionStatus, lastError string, lastFiredAt *time.Time) error {
	if err := trigger.Validate(); err != nil {
		return err
	}
	triggerJSON, err := json.Marshal(trigger)
	if err != nil {
		return fmt.Errorf("marshal trigger: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	var lastFiredAtStr *string
	if lastFiredAt != nil {
		s := lastFiredAt.UTC().Format(time.RFC3339Nano)
		lastFiredAtStr = &s
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO triggers(
			stack_name, name, spec_json, config_hash,
			last_status, last_error, last_fired_at, updated_at, created_at
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(stack_name, name) DO UPDATE SET
			spec_json=excluded.spec_json,
			config_hash=excluded.config_hash,
			last_status=excluded.last_status,
			last_error=excluded.last_error,
			last_fired_at=excluded.last_fired_at,
			updated_at=excluded.updated_at
	`, stackName, trigger.Name, string(triggerJSON), configHash, lastStatus, lastError, lastFiredAtStr, now, now)
	if err != nil {
		return fmt.Errorf("upsert trigger %q: %w", trigger.Name, err)
	}
	return nil
}

func (r *Repository) ListTriggers(ctx context.Context, stackName string) ([]TriggerRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT stack_name, name, spec_json, config_hash, last_status, last_error, last_fired_at, updated_at
		FROM triggers
		WHERE stack_name = ?
		ORDER BY name
	`, stackName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TriggerRecord
	for rows.Next() {
		var rec TriggerRecord
		var triggerJSON string
		var lastFiredAt sql.NullString
		var updatedAt string

		if err := rows.Scan(
			&rec.StackName, &rec.Name, &triggerJSON, &rec.ConfigHash,
			&rec.LastStatus, &rec.LastError, &lastFiredAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(triggerJSON), &rec.Trigger); err != nil {
			return nil, err
		}
		if lastFiredAt.Valid && lastFiredAt.String != "" {
			t, err := time.Parse(time.RFC3339Nano, lastFiredAt.String)
			if err != nil {
				return nil, err
			}
			rec.LastFiredAt = &t
		}
		t, err := time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, err
		}
		rec.UpdatedAt = t

		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *Repository) SaveExecution(ctx context.Context, e models.Execution) error {
	if err := e.Validate(); err != nil {
		return err
	}
	var endedAt any
	if e.EndedAt != nil {
		endedAt = e.EndedAt.UTC().Format(time.RFC3339Nano)
	}
	var exitCode any
	if e.ExitCode != nil {
		exitCode = *e.ExitCode
	}

	metadata := map[string]string{}
	if e.Error != "" {
		metadata["error"] = e.Error
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal execution metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO executions(
			id, workload_kind, workload_name, trigger_name,
			status, started_at, ended_at, exit_code, error, logs_ref, metadata_json
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workload_kind=excluded.workload_kind,
			workload_name=excluded.workload_name,
			trigger_name=excluded.trigger_name,
			status=excluded.status,
			started_at=excluded.started_at,
			ended_at=excluded.ended_at,
			exit_code=excluded.exit_code,
			error=excluded.error,
			logs_ref=excluded.logs_ref,
			metadata_json=excluded.metadata_json
	`, e.ID, e.Kind, e.Workload, e.TriggerName, e.Status, e.StartedAt.UTC().Format(time.RFC3339Nano), endedAt, exitCode, e.Error, e.LogsRef, string(metadataJSON))
	if err != nil {
		return fmt.Errorf("save execution %q: %w", e.ID, err)
	}

	return nil
}

func (r *Repository) ListExecutions(ctx context.Context, workloadKind *models.WorkloadKind, workloadName string, limit int) ([]models.Execution, error) {
	query := `
		SELECT id, workload_kind, workload_name, trigger_name, status, started_at, ended_at, exit_code, error, logs_ref
		FROM executions
		WHERE workload_name = ?
	`
	args := []any{workloadName}

	if workloadKind != nil {
		query += ` AND workload_kind = ?`
		args = append(args, *workloadKind)
	}

	query += ` ORDER BY started_at DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Execution
	for rows.Next() {
		var e models.Execution
		var startedAt string
		var endedAt sql.NullString
		var exitCode sql.NullInt64

		if err := rows.Scan(
			&e.ID, &e.Kind, &e.Workload, &e.TriggerName, &e.Status,
			&startedAt, &endedAt, &exitCode, &e.Error, &e.LogsRef,
		); err != nil {
			return nil, err
		}

		t, err := time.Parse(time.RFC3339Nano, startedAt)
		if err != nil {
			return nil, err
		}
		e.StartedAt = t

		if endedAt.Valid && endedAt.String != "" {
			t2, err := time.Parse(time.RFC3339Nano, endedAt.String)
			if err != nil {
				return nil, err
			}
			e.EndedAt = &t2
		}
		if exitCode.Valid {
			v := int(exitCode.Int64)
			e.ExitCode = &v
		}

		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) SetMetadata(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO runtime_metadata(key, value, updated_at)
		VALUES(?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value=excluded.value,
			updated_at=excluded.updated_at
	`, key, value, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("set metadata %q: %w", key, err)
	}
	return nil
}

func (r *Repository) GetMetadata(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `
		SELECT value FROM runtime_metadata WHERE key = ?
	`, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}