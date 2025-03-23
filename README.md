# kube-proxy

This project can help to proxy dynamic endpoints from a cluster to a local listener (using ssh). It will spin up a ssh jump server, creates a port-forward to it and let's you implement a handler for the incomming `net.Conn`

```golang
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("yehaa\n"))
	})

	p := proxy.New(mux)
	if err := p.Run(context.Background(), cfg, "default"); err != nil {
		panic(err)
	}
```