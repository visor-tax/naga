package service

import (
	"os"
	"os/signal"
	"syscall"
)

// Start is a convenience method equivalent to `service.Load(m).Run()` and starting the
// app with `./<myapp> start`. Prefer using `Run()` as it is more flexible.
func (s *Service) Start() error {
	return s.RunCommand("start")
}

// StartForTest starts the app with the environment set to test.
// Returns stop function as a convenience.
func (s *Service) StartForTest() func() {
	s.Env = EnvTest
	err := s.setup()
	if err != nil {
		panic(err)
	}
	go s.start()
	s.started.Wait()
	return s.Stop
}

// Provide sets up the service synchronoously and then starts it in a goroutine.
// This method is intended as an adapter to Wire.
func (s *Service) Provide() (func(), error) {
	err := s.setup()
	if err != nil {
		return nil, err
	}
	go s.start()
	s.started.Wait()
	return s.Stop, nil
}

// start calls Start on each module, in goroutines. Assumes that
// setup() has already been called. Start command must not block.
func (s *Service) start() {
	for _, m := range s.modules {
		n := getModuleName(m)
		c := s.configs[n]
		BootPrintln("[service] starting", n)
		if c.Start != nil {
			c.Start()
		}
	}
	// mark as started
	s.started.Done()

	// mark process as running
	s.running.Add(1)

	// wait for a stop signal to be received
	// note: that c might crash without this parent goroutine knowing.
	s.wait()

	// mark process as done
	s.running.Done()
}

// wait blocks until a signal is received, or the stopper channel is closed
func (s *Service) wait() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, os.Kill)
	select {
	case sig := <-c:
		BootPrintln("[service] got signal:", sig)
	case <-s.stopper:
		BootPrintln("[service] app stop")
	}
	s.stop()
}
