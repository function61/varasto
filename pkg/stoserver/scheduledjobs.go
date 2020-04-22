package stoserver

import (
	"context"
	"log"
	"time"

	"github.com/function61/eventhorizon/pkg/ehevent"
	"github.com/function61/eventkit/command"
	"github.com/function61/eventkit/eventlog"
	"github.com/function61/eventkit/httpcommand"
	"github.com/function61/varasto/pkg/scheduler"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

// current middlewares has this empty too
const FIXMEsystemUserId = ""

type smartPollerScheduledJob struct {
	commandPlumbing *scheduledJobCommandPlumbing
}

func (s *smartPollerScheduledJob) GetRunner() scheduler.JobFn {
	return commandInvokerJobFn(&stoservertypes.NodeSmartScan{}, s.commandPlumbing)
}

type metadataBackupScheduledJob struct {
	commandPlumbing *scheduledJobCommandPlumbing
}

func (s *metadataBackupScheduledJob) GetRunner() scheduler.JobFn {
	return commandInvokerJobFn(&stoservertypes.DatabaseBackup{}, s.commandPlumbing)
}

func scheduledJobRunner(
	kind stoservertypes.ScheduledJobKind,
	commandPlumbing *scheduledJobCommandPlumbing,
) scheduler.JobFn {
	switch stoservertypes.ScheduledJobKindExhaustive1edcb7(kind) {
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

func setupScheduledJobs(
	invoker command.Invoker,
	eventLog eventlog.Log,
	db *bbolt.DB,
	logger *log.Logger,
	startScheduler func(fn func(context.Context) error),
	startSnapshotter func(fn func(context.Context) error),
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

	schedulerController := scheduler.New(jobs, logger, startScheduler)

	startSnapshotter(func(ctx context.Context) error {
		// when scheduler job's state changes, it emits a snapshot that we'll save to the DB
		for {
			select {
			case <-ctx.Done():
				return nil
			case snapshot, ok := <-schedulerController.SnapshotReady:
				if !ok { // chan closed => graceful stop
					return nil
				}

				if err := handleSnapshot(db, snapshot); err != nil {
					return err // shuts done entire server
				}
			}
		}
	})

	return schedulerController, nil
}

func handleSnapshot(db *bbolt.DB, snapshot []scheduler.JobSpec) error {
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

// data a scheduled job needs to be able to invoke commands
type scheduledJobCommandPlumbing struct {
	invoker  command.Invoker
	eventLog eventlog.Log
}

// makes a scheduled job function that executes a command
func commandInvokerJobFn(
	cmd command.Command,
	commandPlumbing *scheduledJobCommandPlumbing,
) scheduler.JobFn {
	return func(ctx context.Context, logger *log.Logger) error {
		cmdCtx := command.NewCtx(
			ctx,
			ehevent.Meta(time.Now(), FIXMEsystemUserId),
			"",
			"")

		if err := httpcommand.InvokeSkippingAuthorization(
			cmd,
			cmdCtx,
			commandPlumbing.invoker,
			commandPlumbing.eventLog,
		); err != nil {
			return err
		}

		return nil
	}
}
