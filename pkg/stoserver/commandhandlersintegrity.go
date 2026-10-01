package stoserver

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/function61/eventkit/command"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stointegrityverifier"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"go.etcd.io/bbolt"
)

func (c *cHandlers) IntegrityverificationjobResume(cmd *stoservertypes.IntegrityverificationjobResume, ctx *command.Ctx) error {
	if cmd.NextSampleBatch {
		var nextJobID string
		if err := c.db.Update(func(tx *bbolt.Tx) error {
			previousJob := &stotypes.IntegrityVerificationJob{}
			if err := stodb.IntegrityVerificationJobRepository.OpenByPrimaryKey([]byte(cmd.JobId), previousJob, tx); err != nil {
				return err
			}
			if previousJob.SampleSpecification == nil {
				return errors.New("cannot create next sample batch for a job without sampling")
			}

			nextSampleSpecification, wrapped, err := stointegrityverifier.NextSampleSpecification(*previousJob.SampleSpecification)
			if err != nil {
				return err
			}
			if wrapped {
				return fmt.Errorf("sampling specification %q is already the final batch", *previousJob.SampleSpecification)
			}

			nextJobID = stoutils.NewIntegrityVerificationJobID()
			return stodb.IntegrityVerificationJobRepository.Update(&stotypes.IntegrityVerificationJob{
				ID:                  nextJobID,
				Started:             ctx.Meta.Timestamp,
				VolumeID:            previousJob.VolumeID,
				SampleSpecification: &nextSampleSpecification,
			}, tx)
		}); err != nil {
			return err
		}

		ctx.CreatedRecordId(nextJobID)
		c.ivController.Resume(nextJobID)
		return nil
	}

	c.ivController.Resume(cmd.JobId)

	return nil
}

func (c *cHandlers) IntegrityverificationjobStop(cmd *stoservertypes.IntegrityverificationjobStop, ctx *command.Ctx) error {
	c.ivController.Stop(cmd.JobId)

	return nil
}

func (c *cHandlers) IntegrityverificationjobReverify(cmd *stoservertypes.IntegrityverificationjobReverify, ctx *command.Ctx) error {
	if slices.Contains(c.ivController.ListRunningJobs(), cmd.JobId) {
		return fmt.Errorf("cannot re-check problems of job '%s' when it is running right now", cmd.JobId)
	}

	if cmd.BlobRefs == nil || len(*cmd.BlobRefs) == 0 {
		return errors.New("at least one blob reference is required")
	}

	refs := map[string]stotypes.BlobRef{}
	for _, blobRefText := range *cmd.BlobRefs {
		blobRef, err := stotypes.BlobRefFromHex(blobRefText)
		if err != nil {
			return err
		}
		refs[blobRef.AsHex()] = *blobRef
	}

	job := &stotypes.IntegrityVerificationJob{}
	if err := c.db.View(func(tx *bbolt.Tx) error {
		return stodb.IntegrityVerificationJobRepository.OpenByPrimaryKey([]byte(cmd.JobId), job, tx)
	}); err != nil {
		return err
	}

	successfulRefs := map[string]bool{}
	for refText, blobRef := range refs {
		if _, err := c.conf.DiskAccess.Scrub(ctx.Ctx, blobRef, job.VolumeID); err == nil {
			successfulRefs[refText] = true
		} else {
			slog.Error("blob re-verification failed", "job_id", cmd.JobId, "blob_ref", refText, "volume_id", job.VolumeID, "error", err)
		}
	}

	if len(successfulRefs) == 0 {
		return errors.New("none of the selected blobs could be re-verified")
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		job := &stotypes.IntegrityVerificationJob{}
		if err := stodb.IntegrityVerificationJobRepository.OpenByPrimaryKey([]byte(cmd.JobId), job, tx); err != nil {
			return err
		}

		remainingIssues := job.Issues[:0]
		for _, issue := range job.Issues {
			if issue.Blob == nil || !successfulRefs[issue.Blob.AsHex()] {
				remainingIssues = append(remainingIssues, issue)
			}
		}
		job.Issues = remainingIssues

		return stodb.IntegrityVerificationJobRepository.Update(job, tx)
	})
}

func (c *cHandlers) IntegrityverificationjobClearProblems(cmd *stoservertypes.IntegrityverificationjobClearProblems, ctx *command.Ctx) error {
	if slices.Contains(c.ivController.ListRunningJobs(), cmd.JobId) {
		return fmt.Errorf("cannot clear problems of job '%s' when it is running right now", cmd.JobId)
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		job := &stotypes.IntegrityVerificationJob{}
		if err := stodb.IntegrityVerificationJobRepository.OpenByPrimaryKey([]byte(cmd.JobId), job, tx); err != nil {
			return err
		}

		job.Issues = nil

		return stodb.IntegrityVerificationJobRepository.Update(job, tx)
	})
}
