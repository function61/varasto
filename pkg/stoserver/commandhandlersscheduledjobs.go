package stoserver

import (
	"errors"

	"github.com/function61/eventkit/command"
	"github.com/function61/varasto/pkg/scheduler"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

func (c *cHandlers) ScheduledjobEnable(cmd *stoservertypes.ScheduledjobEnable, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		job, err := openScheduledJobNotUpdater(cmd.Id, tx)
		if err != nil {
			return err
		}

		if !job.Enabled {
			job.Enabled = true
		} else {
			return errors.New("job already enabled")
		}

		return stodb.ScheduledJobRepository.Update(job, tx)
	})
}

func (c *cHandlers) ScheduledjobDisable(cmd *stoservertypes.ScheduledjobDisable, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		job, err := openScheduledJobNotUpdater(cmd.Id, tx)
		if err != nil {
			return err
		}

		if job.Enabled {
			job.Enabled = false
		} else {
			return errors.New("job already disabled")
		}

		return stodb.ScheduledJobRepository.Update(job, tx)
	})
}

func (c *cHandlers) ScheduledjobChangeSchedule(cmd *stoservertypes.ScheduledjobChangeSchedule, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		job, err := openScheduledJobNotUpdater(cmd.Id, tx)
		if err != nil {
			return err
		}

		job.Schedule = cmd.Schedule

		if _, err := scheduler.ValidateSpec(dbJobToJobSpec(*job)); err != nil {
			return err
		}

		return stodb.ScheduledJobRepository.Update(job, tx)
	})
}

func (c *cHandlers) ScheduledjobStart(cmd *stoservertypes.ScheduledjobStart, ctx *command.Ctx) error {
	c.conf.Scheduler.Trigger(cmd.Id)

	return nil
}

// this would mess up our analytics
func openScheduledJobNotUpdater(id string, tx *bbolt.Tx) (*stotypes.ScheduledJob, error) {
	job, err := stodb.Read(tx).ScheduledJob(id)
	if err != nil {
		return nil, err
	}

	if job.ID == stoservertypes.UpdateCheckerScheduledJobId {
		return nil, errors.New("editing update checker is disabled")
	}

	return job, nil
}
