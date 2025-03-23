package handler

import (
	"context"
	"fmt"
	"io"
	pipelistener "kube-proxy/pkg/listener"
	"log"
	"net"
	"net/http"

	"golang.org/x/crypto/ssh"
)

type Handler struct {
	mux *http.ServeMux
}

func New(m *http.ServeMux) *Handler {
	return &Handler{m}
}

func (h *Handler) Run(ctx context.Context, port string) error {
	sshConfig := &ssh.ClientConfig{
		User:            "test",
		Auth:            []ssh.AuthMethod{ssh.Password("test")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	serverConn, err := ssh.Dial("tcp", net.JoinHostPort("127.0.0.1", port), sshConfig)
	if err != nil {
		return fmt.Errorf("dial INTO remote server error - %w", err)
	}

	remoteListener, err := serverConn.Listen("tcp", "0.0.0.0:8080")
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

			log.Println("handling new connection")
			if err := pl.ServeConn(client); err != nil {
				log.Printf("failed to serve conn - %s", err.Error())
			}
		}
	}()

	go func() {
		<-ctx.Done()
		log.Println("stopping remote listener")
		remoteListener.Close()
	}()

	return nil
}
