package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	pipelistener "github.com/kaedwen/kube-proxy/pkg/listener"

	"golang.org/x/crypto/ssh"
)

type Handler struct {
	log *log.Logger
	mux *http.ServeMux
}

func New(log *log.Logger, m *http.ServeMux) *Handler {
	return &Handler{log, m}
}

func (h *Handler) Run(ctx context.Context, localhost string, remoteport string) error {
	sshConfig := &ssh.ClientConfig{
		User:            "test",
		Auth:            []ssh.AuthMethod{ssh.Password("test")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	serverConn, err := ssh.Dial("tcp", net.JoinHostPort("127.0.0.1", localhost), sshConfig)
	if err != nil {
		return fmt.Errorf("dial INTO remote server error - %w", err)
	}

	remoteListener, err := serverConn.Listen("tcp", net.JoinHostPort("0.0.0.0", remoteport))
	if err != nil {
		return fmt.Errorf("listen open port ON remote server error - %w", err)
	}

	srv := &http.Server{
		Handler: h.mux,
	}

	pl := pipelistener.New()
	go func() {
		if err := srv.Serve(pl); err != nil {
			panic(err)
		}
	}()

	go func() {
		for {
			client, err := remoteListener.Accept()
			if err != nil {
				if err == io.EOF {
					break
				}

				log.Println(err)
				continue
			}

			h.log.Println("handling new connection")
			if err := pl.ServeConn(client); err != nil {
				h.log.Printf("failed to serve conn - %s", err.Error())
			}
		}
	}()

	go func() {
		<-ctx.Done()
		h.log.Println("stopping remote listener")
		remoteListener.Close()
	}()

	return nil
}
