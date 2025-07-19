package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/function61/gokit/logex"
	"github.com/robfig/cron/v3"
)

type JobLastRun struct {
	Started  time.Time
	Finished time.Time
	Error    string
}

type JobFn func(ctx context.Context, logger *log.Logger) error

type Job struct {
	Spec     JobSpec
	Run      JobFn
	Schedule cron.Schedule
}

var cronParser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

func ValidateSpec(spec JobSpec) (cron.Schedule, error) {
	return cronParser.Parse(spec.Schedule)
}

func NewJob(spec JobSpec, run JobFn, now time.Time) (*Job, error) {
	schedule, err := ValidateSpec(spec)
	if err != nil {
		return nil, err
	}

	if spec.NextRun.IsZero() {
		spec.NextRun = schedule.Next(now)
	}

	return &Job{
		Spec:     spec,
		Run:      run,
		Schedule: schedule,
	}, nil
}

type JobSpec struct {
	ID          string
	Description string
	NextRun     time.Time
	Running     bool
	Schedule    string
	LastRun     *JobLastRun
}

type snapshotRequest struct {
	result chan []JobSpec
}

type jobResult struct {
	job *Job
	run *JobLastRun
}

type Controller struct {
	snapshotRequest chan *snapshotRequest
	triggerRequest  chan string
	jobFinished     chan *jobResult
	SnapshotReady   chan []JobSpec
	jobLogger       *log.Logger
}

func New(
	jobs []*Job,
	jobLogger *log.Logger,
	start func(func(context.Context) error),
) *Controller {
	c := &Controller{
		make(chan *snapshotRequest),
		make(chan string),
		make(chan *jobResult, 1),
		make(chan []JobSpec, 2), // TODO: use mailbox pattern from gokit (or eventhorizon)?
		jobLogger,
	}

	start(func(ctx context.Context) error {
		return c.run(ctx, jobs)
	})

	return c
}

func (s *Controller) Trigger(jobID string) {
	s.triggerRequest <- jobID
}

// gets an atomic snapshot of scheduler's internal state
func (s *Controller) Snapshot() []JobSpec {
	result := make(chan []JobSpec, 1)

	s.snapshotRequest <- &snapshotRequest{result}

	return <-result
}

// the core of the scheduler runs single-threaded, but many interactions like task running and
// requesting snapshot of job state are in other goroutines and communication happens via channels
func (s *Controller) run(ctx context.Context, jobs []*Job) error {
	defer func() {
		close(s.SnapshotReady)
	}()

	nextEarliestCh := func() <-chan time.Time {
		if len(jobs) == 0 {
			return nil // channel that blocks forever
		}

		earliest := jobs[0].Spec.NextRun
		for _, job := range jobs {
			if job.Spec.NextRun.Before(earliest) {
				earliest = job.Spec.NextRun
			}
		}

		return time.After(time.Until(earliest))
	}

	makeSnapshot := func() []JobSpec {
		jobCopies := []JobSpec{}

		for _, job := range jobs {
			jobCopies = append(jobCopies, copyJobSpec(job.Spec))
		}

		return jobCopies
	}

	recordJobFinished := func(jr *jobResult) {
		jr.job.Spec.LastRun = jr.run

		jr.job.Spec.Running = false

		s.SnapshotReady <- makeSnapshot()
	}

	nextJobBecomesRunnableCh := nextEarliestCh()

	for {
		select {
		case now := <-nextJobBecomesRunnableCh:
			for _, job := range jobs {
				if !job.Spec.NextRun.After(now) {
					s.startJob(ctx, job)
				}
			}

			nextJobBecomesRunnableCh = nextEarliestCh()
		case snapshotReq := <-s.snapshotRequest:
			snapshotReq.result <- makeSnapshot()
		case jobResult := <-s.jobFinished:
			recordJobFinished(jobResult)
		case jobID := <-s.triggerRequest:
			for _, job := range jobs {
				if job.Spec.ID == jobID {
					s.startJob(ctx, job)
					break
				}
			}
		case <-ctx.Done():
			for _, job := range jobs {
				if job.Spec.Running {
					// wait for the first of the N running stops to finish - not necessarily
					// the "job" variable we have. it's mainly used to count # of unfinished jobs
					recordJobFinished(<-s.jobFinished)
				}
			}

			return nil // stops scheduler
		}
	}
}

func (s *Controller) startJob(ctx context.Context, job *Job) {
	job.Spec.NextRun = job.Schedule.Next(job.Spec.NextRun)

	jlog := logex.Prefix("scheduler/"+job.Spec.Description, s.jobLogger)
	jlogl := logex.Levels(jlog)

	if job.Spec.Running {
		jlogl.Error.Println("can't start job since previous instance is still running")
		return
	}

	job.Spec.Running = true

	jlogl.Info.Println("starting")

	go func() {
		started := time.Now()

		errorStr := ""
		if err := job.Run(ctx, jlog); err != nil {
			errorStr = err.Error()
		}

		result := &jobResult{
			job: job,
			run: &JobLastRun{
				Started:  started,
				Error:    errorStr,
				Finished: time.Now(),
			},
		}

		duration := result.run.Finished.Sub(result.run.Started)

		if errorStr != "" {
			jlogl.Error.Printf("in %s: %s", duration, errorStr)
		} else {
			jlogl.Info.Printf("completed in %s", duration)
		}

		s.jobFinished <- result
	}()
}

func copyJobSpec(copied JobSpec) JobSpec {
	if copied.LastRun != nil {
		lastRunCopied := *copied.LastRun

		copied.LastRun = &lastRunCopied
	}

	return copied
}
