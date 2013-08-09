package poolhandler

import (
	"io"
	"net/http"
	"strconv"
	"time"
)

type PoolHandler struct {
	handler http.Handler
	reqchan chan poolRequest
}

type poolRequest struct {
	w       http.ResponseWriter
	req     *http.Request
	handler http.Handler
	result  chan bool
}

type poolWorker struct {
	Busy                           bool
	Start, End                     time.Time
	Method, Host, Path, RemoteAddr string
}

func (pw *poolWorker) DoWork(poolreq poolRequest, free chan bool) {
	pw.Method = poolreq.req.Method
	pw.Host = poolreq.req.Host
	pw.Path = poolreq.req.URL.Path
	pw.RemoteAddr = poolreq.req.RemoteAddr
	poolreq.handler.ServeHTTP(poolreq.w, poolreq.req)
	poolreq.result <- true
	pw.Busy = false
	free <- true
}

func poolManager(maxclients int, reqchan chan poolRequest, statusURL string) {
	var poolreq poolRequest
	var totalCount int
	var nowCount int
	free := make(chan bool, 2)
	pool := make([]poolWorker, maxclients, maxclients)
	for {
		if nowCount < maxclients {
			select {
			case poolreq = <-reqchan:
				totalCount++
				nowCount++
				for key := range pool {
					if !pool[key].Busy {
						pool[key].Busy = true
						if poolreq.req.URL.Path != "" && poolreq.req.URL.Path == statusURL {
							pool[key].Method = poolreq.req.Method
							pool[key].Host = poolreq.req.Host
							pool[key].Path = poolreq.req.URL.Path
							pool[key].RemoteAddr = poolreq.req.RemoteAddr
							io.WriteString(poolreq.w, "<html><head></head><body>")
							io.WriteString(poolreq.w, "Total requests: "+strconv.Itoa(totalCount)+"<br />")
							io.WriteString(poolreq.w, "Active requests: "+strconv.Itoa(nowCount)+"/"+strconv.Itoa(maxclients)+"<br />")
							for _, worker := range pool {
								if worker.Busy {
									io.WriteString(poolreq.w, "W")
								} else {
									io.WriteString(poolreq.w, "_")
								}
							}
							io.WriteString(poolreq.w, "<br /><table border=\"0\"><tr>")
							io.WriteString(poolreq.w, "<td><b>#</b></td>")
							io.WriteString(poolreq.w, "<td><b>W</b></td>")
							io.WriteString(poolreq.w, "<td><b>Client</b></td>")
							io.WriteString(poolreq.w, "<td><b>VHost</b></td>")
							io.WriteString(poolreq.w, "<td nowrap><b>Request</b></td></tr>")
							for k, worker := range pool {
								io.WriteString(poolreq.w, "<tr><td><b>"+strconv.Itoa(k)+"</b></td>")
								if worker.Busy {
									io.WriteString(poolreq.w, "<td><b>W</b></td>")
								} else {
									io.WriteString(poolreq.w, "<td>_</td>")
								}
								io.WriteString(poolreq.w, "<td>"+worker.RemoteAddr+"</td>")
								io.WriteString(poolreq.w, "<td>"+worker.Host+"</td>")
								io.WriteString(poolreq.w, "<td nowrap>"+worker.Method+" "+worker.Path+"</td>")
								io.WriteString(poolreq.w, "</tr>")
							}
							io.WriteString(poolreq.w, "</table></body></html>\n")
							pool[key].Busy = false
							nowCount--
							poolreq.result <- true
						} else {
							go pool[key].DoWork(poolreq, free)
						}
						break
					}
				}
			case <-free:
				nowCount--
			}
		} else {
			<-free
			nowCount--
		}
	}
}

func (h PoolHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	result := make(chan bool)
	h.reqchan <- poolRequest{w, req, h.handler, result}
	<-result
}

func GetPoolHandler(maxclients int, h http.Handler, statusURL string) http.Handler {
	reqchan := make(chan poolRequest)
	go poolManager(maxclients, reqchan, statusURL)
	return PoolHandler{h, reqchan}
}
