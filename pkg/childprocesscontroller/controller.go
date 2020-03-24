// Represents a child process whose state we want to control (start, stop, keep alive after crashes),
package childprocesscontroller

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/logtee"
)

type Status struct {
	Description string
	Pid         string
	Alive       bool
	Started     time.Time
}

type Controller struct {
	cmd           []string
	description   string
	status        *Status
	statusMu      sync.Mutex
	start         chan interface{}
	stop          chan interface{}
	exited        chan error
	controlLogger *logex.Leveled
	logger        *log.Logger // subprocess's stderr is logger.Println()'d here after per each line
}

func New(
	cmd []string,
	description string,
	controlLogger *log.Logger,
	logger *log.Logger,
	stop *stopper.Stopper,
) *Controller {
	proc := &Controller{
		cmd:           cmd,
		description:   description,
		start:         make(chan interface{}),
		stop:          make(chan interface{}),
		exited:        make(chan error, 1),
		controlLogger: logex.Levels(controlLogger),
		logger:        logger,
	}

	go proc.handler(stop)

	return proc
}

// means not necessarily starting a single process, but instead that we'll want to keep
// this subprocess alive. if it dies, it might get restarted automatically, in which case
// it gets a new pid, new start time etc.
func (s *Controller) Start() {
	s.start <- nil
}

// same goes as for start. stop means that we'll want the subprocess to stop (or "pause").
// do not call this when you want to gracefully shut down your app, but instead use the
// stopper mechanism which will automatically tear things down gracefully
func (s *Controller) Stop() {
	s.stop <- nil
}

func (s *Controller) Status() Status {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()

	if s.status == nil {
		return Status{
			Description: s.description,
			Alive:       false,
		}
	}

	return *s.status
}

func (s *Controller) setStatus(st *Status) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()

	s.status = st
}

func (s *Controller) handler(stop *stopper.Stopper) {
	defer stop.Done()

	var cmd *exec.Cmd

	desiredRunning := false
	isRunning := func() bool {
		return cmd != nil
	}

	stopSubprocess := func() {
		s.controlLogger.Info.Printf("interrupting pid %d", cmd.Process.Pid)

		// TODO: interrupt does not work on Windows
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			s.controlLogger.Error.Printf("Signal(): %v", err)
		}

		if err := <-s.exited; err != nil {
			s.controlLogger.Error.Printf("unclean exit: %v", err)
		} else {
			s.controlLogger.Info.Println("stopped")
		}

		cmd = nil
		s.setStatus(nil)
	}

	startChildProcess := func() {
		cmd = exec.Command(s.cmd[0], s.cmd[1:]...)
		// child should receive full env of parent
		cmd.Env = append(cmd.Env, os.Environ()...)
		// TODO: is it bad if this key possibly is duplicate?
		cmd.Env = append(cmd.Env, "LOGGER_SUPPRESS_TIMESTAMPS=1")

		// TODO: what about stdout?

		cmd.Stderr = logtee.NewLineSplitterTee(ioutil.Discard, func(line string) {
			s.logger.Println(line)
		})

		// open stdin that does nothing, so that subprocess can detect closure of its
		// stdin to mean that its parent process has died disgracefully
		_, err := cmd.StdinPipe()
		if err != nil {
			s.exited <- err
			return
		}

		if err := cmd.Start(); err != nil {
			s.exited <- err
			return
		}

		s.controlLogger.Info.Printf("started (pid %d)", cmd.Process.Pid)

		s.setStatus(&Status{
			Description: s.description,
			Pid:         strconv.Itoa(cmd.Process.Pid),
			Alive:       true,
			Started:     time.Now(),
		})

		go func() {
			s.exited <- cmd.Wait()
		}()
	}

	for {
		select {
		case <-s.start:
			desiredRunning = true

			if !isRunning() {
				startChildProcess()
			}
		case <-stop.Signal:
			if isRunning() {
				stopSubprocess()
			}

			// stops this handler goroutine & reports completion via stopper
			return
		case <-s.stop:
			desiredRunning = false

			if isRunning() {
				stopSubprocess()
			}
		case err := <-s.exited:
			cmd = nil
			s.setStatus(nil)

			if desiredRunning {
				dur := 5 * time.Second

				s.controlLogger.Error.Printf(
					"unexpected exit with %v (restarting in %s)",
					err,
					dur)

				time.Sleep(dur)

				startChildProcess()
			} else {
				s.controlLogger.Debug.Printf("process %s expectedly exited with %v", s.description, err)
			}
		}
	}
}
