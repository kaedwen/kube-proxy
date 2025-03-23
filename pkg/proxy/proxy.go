package proxy

import (
	"context"
	"errors"
	"kube-proxy/pkg/forwarder"
	"kube-proxy/pkg/handler"
	"kube-proxy/pkg/pod"
	"net/http"
	"strconv"
	"time"

	"k8s.io/client-go/rest"
)

type KubeProxy struct {
	mux *http.ServeMux
	ph  *pod.Handler
}

func New(mux *http.ServeMux) *KubeProxy {
	return &KubeProxy{mux: mux}
}

func (p *KubeProxy) Run(ctx context.Context, cfg *rest.Config, namespace string) error {
	if p.ph != nil {
		return errors.New("already running")
	}

	var err error
	p.ph, err = pod.New(cfg, namespace)
	if err != nil {
		return err
	}

	if err := p.ph.Start(ctx); err != nil {
		return err
	}

	// wait before forwarder spin-up
	<-time.After(5 * time.Second)

	port, err := forwarder.New(cfg, namespace).Run(ctx, p.ph)
	if err != nil {
		return err
	}

	hnd := handler.New(p.mux)
	if err := hnd.Run(ctx, strconv.Itoa(port)); err != nil {
		return err
	}

	return nil
}

func (p *KubeProxy) Cleanup(ctx context.Context) error {
	return p.ph.Delete(ctx)
}
