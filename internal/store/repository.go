package store

import "github.com/duku/net-lab/internal/model"
import "time"

type Repository interface {
	Status() model.Status
	Sessions() []model.CaptureSession
	Radios() []model.AuthorizedRadio
	Schedules() []model.CaptureSchedule
	Metrics() []model.MetricBucket
	Devices() []model.Device
	Findings() []model.Finding
	RouterSamples() []model.RouterSample
	AddRadio(model.AuthorizedRadio) model.AuthorizedRadio
	AddSchedule(model.CaptureSchedule) (model.CaptureSchedule, error)
	StartCapture(int) model.HostCommand
	StopCapture() model.HostCommand
	NextHostCommand() (model.HostCommand, error)
	CompleteHostCommand(string, string) error
	Ingest([]model.MetricBucket, []model.Finding)
	TickSchedules(time.Time) []model.HostCommand
	SeedDemo()
	ResetDemo()
}
