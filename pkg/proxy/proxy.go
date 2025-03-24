package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/kaedwen/kube-proxy/pkg/forwarder"
	"github.com/kaedwen/kube-proxy/pkg/handler"
	"github.com/kaedwen/kube-proxy/pkg/pod"

	"k8s.io/client-go/rest"
)

const (
	prefix = "kube-proxy: "
)

type KubeProxy struct {
	log *log.Logger
	hnd *handler.Handler
	ph  *pod.Handler
}

func New(options ...ProxyPotion) *KubeProxy {
	// create default logger
	log := log.New(os.Stdout, prefix, 0)

	p := &KubeProxy{log: log, hnd: handler.New(log), ph: nil}

	// apply the options
	for _, opt := range options {
		opt(p)
	}

	return p
}

func (p *KubeProxy) SetMux(mux *http.ServeMux) {
	p.hnd.SetMux(mux)
}

func (p *KubeProxy) Run(ctx context.Context, cfg *rest.Config, namespace string, remoteport string) error {
	if p.ph != nil {
		return errors.New("already running")
	}

	var err error
	p.ph, err = pod.New(p.log, cfg, namespace)
	if err != nil {
		return err
	}

	if err := p.ph.Start(ctx); err != nil {
		return err
	}

	// wait before forwarder spin-up
	<-time.After(5 * time.Second)

	port, err := forwarder.New(p.log, cfg, namespace).Run(ctx, p.ph)
	if err != nil {
		return err
	}

	if err := p.hnd.Run(ctx, strconv.Itoa(port), remoteport); err != nil {
		return err
	}

	return nil
}

func (p *KubeProxy) Endpoint() string {
	if p.ph == nil {
		return ""
	}

	return fmt.Sprintf("%s.%s.pod.cluster.local", p.ph.Name(), p.ph.Namespace())
}

func (p *KubeProxy) Cleanup(ctx context.Context) error {
	return p.ph.Delete(ctx)
}

func (p *KubeProxy) Teardown() {
	p.ph.Delete(context.Background())
}

type ProxyPotion func(p *KubeProxy)

func WithMux(mux *http.ServeMux) ProxyPotion {
	return func(p *KubeProxy) {
		p.hnd.SetMux(mux)
	}
}

func WithLoggerOutput(w io.Writer) ProxyPotion {
	return func(p *KubeProxy) {
		p.log.SetOutput(w)
	}
}
