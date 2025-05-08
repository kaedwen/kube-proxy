package forwarder

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path"
	"time"

	utils "github.com/kaedwen/kube-proxy/pkg"
	"github.com/kaedwen/kube-proxy/pkg/pod"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type Forwarder struct {
	log          *log.Logger
	cfg          *rest.Config
	ns           string
	stopChannel  chan struct{}
	readyChannel chan struct{}
}

func New(log *log.Logger, cfg *rest.Config, ns string) *Forwarder {
	return &Forwarder{
		log:          log,
		cfg:          cfg,
		ns:           ns,
		stopChannel:  make(chan struct{}),
		readyChannel: make(chan struct{}),
	}
}

func (f *Forwarder) Run(ctx context.Context, ph *pod.Handler) (int, error) {
	c, err := kubernetes.NewForConfig(f.cfg)
	if err != nil {
		return 0, err
	}

	targetURL := c.RESTClient().Post().
		Resource("pods").
		Namespace(ph.Namespace()).
		Name(ph.Name()).
		SubResource("portforward").URL()

	targetURL.Path = path.Join(
		"api", "v1",
		"namespaces", ph.Namespace(),
		"pods", ph.Name(),
		"portforward",
	)

	transport, upgrader, err := spdy.RoundTripperFor(f.cfg)
	if err != nil {
		return 0, err
	}

	f.log.Println(targetURL.String())

	p, err := utils.GetFreePort()
	if err != nil {
		return 0, err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, targetURL)

	pf, err := portforward.NewOnAddresses(dialer, []string{"0.0.0.0"}, []string{fmt.Sprintf("%d:2222", p)}, f.stopChannel, f.readyChannel, f.log.Writer(), f.log.Writer())
	if err != nil {
		return 0, err
	}

	go func() {
		if err := pf.ForwardPorts(); err != nil {
			f.log.Println("forward", err)
		}
	}()

	<-pf.Ready

	go func() {
		<-ctx.Done()
		<-time.After(5 * time.Second)
		f.log.Println("stopping forwarder")
		close(f.stopChannel)
		pf.Close()
	}()

	return p, nil
}
