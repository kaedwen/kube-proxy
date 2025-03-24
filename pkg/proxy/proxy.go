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

type KubeProxy struct {
	mux *http.ServeMux
	log *log.Logger
	ph  *pod.Handler
}

func New(mux *http.ServeMux) *KubeProxy {
	return &KubeProxy{mux: mux, log: log.New(os.Stdout, "kube-proxy", 0)}
}

func NewWithLoggerOutput(w io.Writer, mux *http.ServeMux) *KubeProxy {
	return &KubeProxy{mux: mux, log: log.New(w, "kube-proxy", 0)}
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

	hnd := handler.New(p.log, p.mux)
	if err := hnd.Run(ctx, strconv.Itoa(port), remoteport); err != nil {
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
