package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"web_security_hw1/pkg/connector"

	"github.com/jmoiron/sqlx"
)

type Proxy struct {
	DB     *sqlx.DB
	Server *http.Server
}

func InitProxy(c *connector.Connector, addr string) (*Proxy, error) {
	proxy := Proxy{}

	proxy.Server = &http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				proxy.HandleTunneling(w, r)
			} else {
				proxy.HandleHTTP(w, r)
			}
		}),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	db_conn, err := c.OpenDB()
	if err != nil {
		return nil, err
	}

	proxy.DB = db_conn
	return &proxy, nil
}

func (p *Proxy) HandleTunneling(w http.ResponseWriter, r *http.Request) {
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go p.transfer(dest_conn, client_conn)
	go p.transfer(client_conn, dest_conn)
}

func (p *Proxy) transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func (p *Proxy) HandleHTTP(w http.ResponseWriter, req *http.Request) {
	err := p.InsertRequest(req, req.RequestURI)
	if err != nil {
		fmt.Printf("HandleHTTP %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		fmt.Printf("HandleHTTP %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	println("handleHTTP")
	p.copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *Proxy) copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

const insertRequest = `
INSERT INTO requests(method, uri, proto) VALUES($1, $2, $3) RETURNING id`

const insertHeader = `
INSERT INTO headers(req_id, key, value) VALUES($1, $2, $3)`

func (p *Proxy) InsertRequest(r *http.Request, uri string) error {
	var id int
	err := p.DB.QueryRow(insertRequest, r.Method, uri, r.Proto).Scan(&id)
	if err != nil {
		fmt.Printf("InsertRequest %s\n", err.Error())
		return err
	}

	for key, value := range r.Header {
		_, err := p.DB.Exec(insertHeader, id, key, value[0])
		if err != nil {
			fmt.Printf("InsertRequest %s\n", err.Error())
			return err
		}
	}

	return nil
}
