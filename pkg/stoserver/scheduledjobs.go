package stoserver

import (
	"context"
	"github.com/function61/eventhorizon/pkg/ehevent"
	"github.com/function61/eventkit/command"
	"github.com/function61/eventkit/eventlog"
	"github.com/function61/eventkit/httpcommand"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/scheduler"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
	"log"
	"time"
)

// current middlewares has this empty too
const FIXMEsystemUserId = ""

type scheduledJobCommandPlumbing struct {
	invoker  command.Invoker
	eventLog eventlog.Log
}

type smartPollerScheduledJob struct {
	commandPlumbing *scheduledJobCommandPlumbing
}

func (s *smartPollerScheduledJob) GetRunner() scheduler.JobFn {
	return func(ctx context.Context, logger *log.Logger) error {
		cmdCtx := command.NewCtx(
			ctx,
			ehevent.Meta(time.Now(), FIXMEsystemUserId),
			"",
			"")

		if err := httpcommand.InvokeSkippingAuthorization(
			&stoservertypes.NodeSmartScan{},
			cmdCtx,
			s.commandPlumbing.invoker,
			s.commandPlumbing.eventLog,
		); err != nil {
			return err
		}

		return nil
	}
}

type metadataBackupScheduledJob struct {
	commandPlumbing *scheduledJobCommandPlumbing
}

func (s *metadataBackupScheduledJob) GetRunner() scheduler.JobFn {
	return func(ctx context.Context, logger *log.Logger) error {
		cmdCtx := command.NewCtx(
			ctx,
			ehevent.Meta(time.Now(), FIXMEsystemUserId),
			"",
			"")

		if err := httpcommand.InvokeSkippingAuthorization(
			&stoservertypes.DatabaseBackup{},
			cmdCtx,
			s.commandPlumbing.invoker,
			s.commandPlumbing.eventLog,
		); err != nil {
			return err
		}

		return nil
	}
}

func scheduledJobRunner(kind stoservertypes.ScheduledJobKind, commandPlumbing *scheduledJobCommandPlumbing) scheduler.JobFn {
	switch stoservertypes.ScheduledJobKindExhaustive89a75e(kind) {
	case stoservertypes.ScheduledJobKindSmartpoll:
		return (&smartPollerScheduledJob{commandPlumbing}).GetRunner()
	case stoservertypes.ScheduledJobKindMetadatabackup:
		return (&metadataBackupScheduledJob{commandPlumbing}).GetRunner()
	// case "user.cmd":
	// case "user.docker":
	default:
		panic("unknown kind: " + kind)
	}
}

// FIXME: these stoppers are not handled properly if we have error setting up scheduler
func setupScheduledJobs(
	invoker command.Invoker,
	eventLog eventlog.Log,
	db *bbolt.DB,
	logger *log.Logger,
	stop *stopper.Stopper,
	snapshotHandlerStop *stopper.Stopper,
) (*scheduler.Controller, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	commandPlumbing := &scheduledJobCommandPlumbing{
		invoker:  invoker,
		eventLog: eventLog,
	}

	dbJobs := []stotypes.ScheduledJob{}
	jobs := []*scheduler.Job{}

	if err := stodb.ScheduledJobRepository.Each(stodb.ScheduledJobAppender(&dbJobs), tx); err != nil {
		return nil, err
	}

	now := time.Now()

	for _, dbJob := range dbJobs {
		if !dbJob.Enabled {
			continue
		}

		job, err := scheduler.NewJob(
			dbJobToJobSpec(dbJob),
			scheduledJobRunner(dbJob.Kind, commandPlumbing),
			now)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	controller, err := scheduler.Start(jobs, logger, stop)
	if err != nil {
		return nil, err
	}

	handleSnapshot := func(snapshot []scheduler.JobSpec) error {
		return db.Update(func(tx *bbolt.Tx) error {
			for _, job := range snapshot {
				dbJob, err := stodb.Read(tx).ScheduledJob(job.Id)
				if err != nil {
					return err
				}

				dbJob.NextRun = job.NextRun
				dbJob.LastRun = convertLastRunToDb(job.LastRun)

				if err := stodb.ScheduledJobRepository.Update(dbJob, tx); err != nil {
					return err
				}
			}

			return nil
		})
	}

	go func() {
		defer snapshotHandlerStop.Done()

		for {
			select {
			case <-snapshotHandlerStop.Signal:
				return
			case snapshot, ok := <-controller.SnapshotReady:
				if !ok {
					return
				}

				if err := handleSnapshot(snapshot); err != nil {
					panic(err)
				}
			}
		}
	}()

	return controller, nil
}

func dbJobToJobSpec(dbJob stotypes.ScheduledJob) scheduler.JobSpec {
	var lastRun *scheduler.JobLastRun
	if dbJob.LastRun != nil {
		lastRun = &scheduler.JobLastRun{
			Started:  dbJob.LastRun.Started,
			Finished: dbJob.LastRun.Finished,
			Error:    dbJob.LastRun.Error,
		}
	}

	return scheduler.JobSpec{
		Id:          dbJob.ID,
		Description: dbJob.Description,
		Schedule:    dbJob.Schedule,
		LastRun:     lastRun,
	}
}

func convertLastRunToDb(lastRun *scheduler.JobLastRun) *stotypes.ScheduledJobLastRun {
	if lastRun == nil {
		return nil
	}

	return &stotypes.ScheduledJobLastRun{
		Started:  lastRun.Started,
		Finished: lastRun.Finished,
		Error:    lastRun.Error,
	}
}
