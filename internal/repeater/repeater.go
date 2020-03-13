package repeater

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
	"web_security_hw1/pkg/connector"

	"github.com/gorilla/mux"
	"github.com/jackc/pgtype"
	"github.com/jmoiron/sqlx"
)

type Request struct {
	Method string `json:"method"`
	Uri    string `json:"uri"`
	Proto  string `json:"proto"`
	Body   string `json:"body,omitempty" `
	Id     int    `json:"id"`
}

type requestSlice struct {
	Res []Request `json:"res"`
}

type Repeater struct {
	DB     *sqlx.DB
	Server *http.Server
	Router *mux.Router
	Schema string
}

func InitRepeater(c *connector.Connector, addr string, schema string) (*Repeater, error) {
	if schema != "http" && schema != "https" {
		return nil, fmt.Errorf("invalid schema")
	}
	repeater := Repeater{Schema: schema}

	repeater.Router = mux.NewRouter()
	repeater.Router.HandleFunc("/{id:[0-9]+}", repeater.RepeatRequest)
	repeater.Router.HandleFunc("/requests", repeater.ShowRequests)

	repeater.Server = &http.Server{
		Addr:    addr,
		Handler: repeater.Router,
	}

	db_conn, err := c.OpenAndCreateDB()
	if err != nil {
		return nil, err
	}

	repeater.DB = db_conn
	return &repeater, nil
}

func (r *Repeater) ShowRequests(w http.ResponseWriter, req *http.Request) {
	result := make([]Request, 0, 1024)
	rowsRequest, err := r.DB.Query("select id, method, uri, proto, body from requests where sch = $1", r.Schema)
	if err != nil {
		fmt.Printf("ShowRequests %s\n", err.Error())
	}

	data := Request{}
	var method string
	var uri string
	var proto string
	var body pgtype.Text
	var id int
	for rowsRequest.Next() {
		err := rowsRequest.Scan(
			&id,
			&method,
			&uri,
			&proto,
			&body,
		)
		data.Id = id
		data.Uri = uri
		data.Method = method
		data.Proto = proto
		data.Body = body.String

		if err != nil {
			fmt.Printf("ShowRequests %s\n", err.Error())
		}
		result = append(result, data)
	}

	bytes, err := json.Marshal(result)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
}

func (r *Repeater) RepeatRequest(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	var method, uri, proto string
	var body pgtype.Text
	var key, value string

	rowsRequest, err := r.DB.Query("select method, uri, proto, body from requests where id = $1 and sch = $2;",
		id, r.Schema)
	if err != nil {
		fmt.Printf("RepeatRequest %s\n", err.Error())
		return
	}

	for rowsRequest.Next() {
		err := rowsRequest.Scan(
			&method,
			&uri,
			&proto,
			&body,
		)
		if err != nil {
			fmt.Printf("RepeatRequest %s\n", err.Error())
			return
		}
	}
	req, err = http.NewRequest(method, uri, strings.NewReader(body.String))
	if err != nil {
		fmt.Printf("RepeatRequest %s\n", err.Error())
		return
	}

	rowsHeaders, err := r.DB.Query("select key, value from headers where req_id = $1;", id)
	if err != nil {
		fmt.Printf("RepeatRequest %s\n", err.Error())
		return
	}
	for rowsHeaders.Next() {
		_ = rowsHeaders.Scan(
			&key,
			&value,
		)

		if key != "If-None-Match" && key != "Accept-Encoding" && key != "If-Modified-Since" {
			req.Header.Add(key, value)
		}
	}

	if r.Schema == "http" {
		HandleHTTP(w, req)
	} else if r.Schema == "https" {
		HandleTunneling(w, req)
	}
}

func HandleHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func HandleTunneling(w http.ResponseWriter, r *http.Request) {
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
	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}
