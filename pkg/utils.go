package utils

import (
	"errors"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func ConfigOrDie() *rest.Config {
	value := os.Getenv("KUBECONFIG")
	data, err := os.ReadFile(value)
	if err != nil {
		panic(err)
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig(data)
	if err != nil {
		panic(err)
	}

	return cfg
}

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

func SigHandler(q chan bool) {
	var quit bool

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// foreach signal received
	for signal := range c {
		// logEvent(lognotice, sys, "Signal received: "+signal.String())

		switch signal {
		case syscall.SIGINT, syscall.SIGTERM:
			quit = true
		case syscall.SIGHUP:
			quit = false
		}

		if quit {
			quit = false
			// closeDb()
			// logEvent(loginfo, sys, "Terminating..")
			// closeLog()
			os.Exit(0)
		}
		// report the value of quit via the channel
		q <- quit
	}
}

func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) error {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return nil // completed normally
	case <-time.After(timeout):
		return errors.New("timeout reached waiting for group")
	}
}
