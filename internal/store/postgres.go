package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/duku/net-lab/internal/model"
)

type Postgres struct {
	db     *sql.DB
	memory *Memory
}

func NewPostgres(databaseURL string) (*Postgres, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS app_state (id text PRIMARY KEY, payload jsonb NOT NULL, updated_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		_ = db.Close()
		return nil, err
	}
	repository := &Postgres{db: db, memory: NewMemory()}
	var payload []byte
	err = db.QueryRow(`SELECT payload FROM app_state WHERE id = 'primary'`).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return repository, nil
	}
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("decode app state: %w", err)
	}
	repository.memory.Restore(snapshot)
	return repository, nil
}

func (p *Postgres) persist() error {
	payload, err := json.Marshal(p.memory.Snapshot())
	if err != nil {
		return err
	}
	_, err = p.db.Exec(`INSERT INTO app_state (id, payload, updated_at) VALUES ('primary', $1, now()) ON CONFLICT (id) DO UPDATE SET payload = EXCLUDED.payload, updated_at = now()`, payload)
	return err
}

func (p *Postgres) Status() model.Status                { return p.memory.Status() }
func (p *Postgres) Sessions() []model.CaptureSession    { return p.memory.Sessions() }
func (p *Postgres) Radios() []model.AuthorizedRadio     { return p.memory.Radios() }
func (p *Postgres) Schedules() []model.CaptureSchedule  { return p.memory.Schedules() }
func (p *Postgres) Metrics() []model.MetricBucket       { return p.memory.Metrics() }
func (p *Postgres) Devices() []model.Device             { return p.memory.Devices() }
func (p *Postgres) Findings() []model.Finding           { return p.memory.Findings() }
func (p *Postgres) RouterSamples() []model.RouterSample { return p.memory.RouterSamples() }
func (p *Postgres) AddRadio(value model.AuthorizedRadio) model.AuthorizedRadio {
	value = p.memory.AddRadio(value)
	_ = p.persist()
	return value
}
func (p *Postgres) AddSchedule(value model.CaptureSchedule) (model.CaptureSchedule, error) {
	value, err := p.memory.AddSchedule(value)
	if err == nil {
		err = p.persist()
	}
	return value, err
}
func (p *Postgres) StartCapture(channel int) model.HostCommand {
	value := p.memory.StartCapture(channel)
	_ = p.persist()
	return value
}
func (p *Postgres) StopCapture() model.HostCommand {
	value := p.memory.StopCapture()
	_ = p.persist()
	return value
}
func (p *Postgres) NextHostCommand() (model.HostCommand, error) { return p.memory.NextHostCommand() }
func (p *Postgres) CompleteHostCommand(id, result string) error {
	if err := p.memory.CompleteHostCommand(id, result); err != nil {
		return err
	}
	return p.persist()
}
func (p *Postgres) Ingest(metrics []model.MetricBucket, findings []model.Finding) {
	p.memory.Ingest(metrics, findings)
	_ = p.persist()
}
func (p *Postgres) TickSchedules(now time.Time) []model.HostCommand {
	commands := p.memory.TickSchedules(now)
	if len(commands) > 0 {
		_ = p.persist()
	}
	return commands
}
func (p *Postgres) SeedDemo()  { p.memory.SeedDemo(); _ = p.persist() }
func (p *Postgres) ResetDemo() { p.memory.ResetDemo(); _ = p.persist() }
