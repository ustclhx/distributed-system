package http

//
// Http server library.
//
// Support concurrent and keep-alive http requests.
// Not support: chuck transfer encoding.
//
// Note:
// * Server use keep-alive http connections regardless of
//   "Connection: keep-alive" header.
// * Content-Length and Host headers are necessary in requests.
// * Content-Length header is necessary in responses.
// * Header value is single.
// * Request-URI must be absolute path. Like: "/add", "/incr".

import (
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"strconv"
	"sync"
	"strings"
	"bytes"
)

// Server here resembles ServeMux in golang standard lib.
// Refer to https://golang.org/pkg/net/http/#ServeMux.
type Server struct {
	Addr     *net.TCPAddr
	l        *net.TCPListener
	mu       sync.Mutex
	doneChan chan struct{}

	// Your data here.
	handlers map[string]Handler
	conns map[*httpConn]struct{}
}

// NewServer initilizes the server of the speficif host.
// The host param includes the hostname and port.
func NewServer(host string) (s *Server) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return nil
	}
	srv := &Server{Addr: tcpAddr}
	srv.doneChan = make(chan struct{})

	// Your initialization code here.
	srv.handlers = make(map[string]Handler)
	srv.conns =make(map[*httpConn]struct{})
	return srv
}

// Handler process the HTTP request and get the response.
//
// Handler should not modify the request.
type Handler interface {
	ServeHTTP(resp *Response, req *Request)
}

// A HandlerFunc responds to an HTTP request.
// Behave the same as he Handler.
type HandlerFunc func(resp *Response, req *Request)

// ServeHTTP calls f(w, r).
func (f HandlerFunc) ServeHTTP(w *Response, r *Request) {
	f(w, r)
}

// NotFoundHandler gives 404 with the blank content.
var NotFoundHandler HandlerFunc = func(resp *Response, req *Request) {
	resp.Write([]byte{})
	resp.WriteStatus(StatusNotFound)
}

// AddHandlerFunc add handlerFunc to the list of handlers.
func (srv *Server) AddHandlerFunc(pattern string, handlerFunc HandlerFunc) {
	srv.AddHandler(pattern, handlerFunc)
}

// AddHandler add handler to the list of handlers.
//
// "" pattern or nil handler is forbidden.
func (srv *Server) AddHandler(pattern string, handler Handler) {
	if pattern == "" {
		panic("http: invalid pattern " + pattern)
	}
	if handler == nil {
		panic("http: nil handler")
	}

	// TODO
    srv.handlers[pattern]=handler
}

// Find a handler matching the path using most-specific
// (longest) matching. If no handler matches, return
// the NotFoundHandler.
func (srv *Server) match(path string) (h Handler) {
	// TODO
	h,ok:=srv.handlers[path]
	if ok {
		return 
	}
	var length int =0
	for pattern,handler := range srv.handlers{
		if !pathMatch(pattern,path){
			continue
		}
		if len(pattern)>length{
			h=handler
			length = len(pattern)
		}
	}
	if h==nil{
		h=NotFoundHandler
	}
	return
}

// Does path match pattern?
// "/" matches path: "/*"
// "/cart/" matches path: "/cart/*"
// "/login" only matches path: "/login"
func pathMatch(pattern, path string) bool {
	if len(pattern) == 0 {
		// should not happen
		return false
	}
	n := len(pattern)
	if pattern[n-1] != '/' {
		return pattern == path
	}
	return len(path) >= n && path[0:n] == pattern
}

// Close immediately closes active net.Listener and any
// active http connections.
//
// Close returns any error returned from closing the Server's
// underlying Listener.
func (srv *Server) Close() (err error) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	select {
	case <-srv.doneChan:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by s.mu.
		close(srv.doneChan)
	}
	err = srv.l.Close()

	// TODO
	for hc:=range srv.conns{
		hc.tcpConn.Close()
		delete(srv.conns, hc)
	}
	
	return
}

// ErrServerClosed is returned by the Server's Serve, ListenAndServe,
// and ListenAndServeTLS methods after a call to Shutdown or Close.
var ErrServerClosed = errors.New("http: Server closed")

// ListenAndServe start listening and serve http connections.
// The method won't return until someone closes the server.
func (srv *Server) ListenAndServe() (err error) {
	// listen on the specific tcp addr, then call Serve()
	l, err := net.ListenTCP("tcp", srv.Addr)
	defer l.Close()
	if err != nil {
		return
	}
	srv.l = l
	// TODO
	for{
		tcpconn,e := l.AcceptTCP()
		if e!=nil{
			select{
				case <-srv.doneChan:
					return ErrServerClosed
				default:
		}
			return e
		}
		hc:=srv.newConn(tcpconn)
		srv.mu.Lock()
		srv.conns[hc]= struct{}{}
		srv.mu.Unlock()
		go hc.serve()
	}
	return
}

func (srv *Server) newConn(conn *net.TCPConn) *httpConn {
	return &httpConn{srv: srv, tcpConn: conn}
}

// A httpConn represents an HTTP connection in the server side.
type httpConn struct {
	srv     *Server
	tcpConn *net.TCPConn
}

// Serve a new connection.
func (hc *httpConn) serve() {
	// Server the http connection in loop way until something goes wrong.
	// The http connection will be closed in the case of errors.
	for {
		// Construct the request from the TCP stream.
		req, err := hc.constructReq()
		if err != nil {
			hc.close()
			return
		}
		resp := &Response{Proto: HTTPVersion, Header: make(map[string]string)}
		
		// Find the matched handler.
		handler := hc.srv.match(req.URL.Path)

		// Handler it in user-defined logics or NotFoundHandler.
		handler.ServeHTTP(resp, req)

		// The response must contain HeaderContentLength in its Header.
		resp.Header[HeaderContentLength] = strconv.FormatInt(resp.ContentLength, 10)

		// *** Discard rest of request body.
			io.Copy(ioutil.Discard, req.Body)
		// Write the response to the TCP stream.
		err = hc.writeResp(resp)
		if err != nil {
			hc.close()
			return
		}
	}
}

// Close the http connection.
func (hc *httpConn) close() {
	// TODO
	hc.srv.mu.Lock()
	defer hc.srv.mu.Unlock()
	hc.tcpConn.Close()
	delete(hc.srv.conns,hc)
}

// Write the response to the TCP stream.
//
// If TCP errors occur, err is not nil.
func (hc *httpConn) writeResp(resp *Response) (err error) {
	// TODO
	var bytes []byte
	bytes=[]byte(resp.Proto+" "+strconv.FormatInt(int64(resp.StatusCode),10)+" "+resp.Status+"\r\n")
	if _,err=hc.tcpConn.Write(bytes);err!=nil{
		return
	}
	for head,value := range resp.Header{
		bytes=[]byte(head+":"+value+"\r\n")
		if _,err=hc.tcpConn.Write(bytes);err!=nil{
			return
		}
	}
	if _,err=hc.tcpConn.Write([]byte("\r\n"));err!=nil{
		return
	}
	if resp.ContentLength!=0{
		if _,err = hc.tcpConn.Write(resp.writeBuff);err!=nil{
			return
		}
	}
	return
}

// Construct the request from the TCP stream.
//
// If TCP errors occur, err is not nil and req is nil.
// Request header must contain the Content-Length.
func (hc *httpConn) constructReq() (req *Request, err error) {
	// TODOr
	req = new(Request)
	var reqline string
	if reqline,err = hc.ReadString('\r');err!=nil{
		return nil,err
	}
	var ok bool
	n :=make([]byte,1)
	req.Method,req.URL,req.Proto,ok = parseRequestLine(reqline)
	if !ok{
		return nil,&HttpError{"WrongRequsetScheme",reqline}
	}
	if req.Method !=MethodGet && req.Method !=MethodPost && req.Method !=MethodPut && req.Method !=MethodPatch{
		return nil,&HttpError{"WrongMethod",req.Method}
	}
	if req.Proto != HTTPVersion{
		return nil,&HttpError{"WrongVersion",req.Proto}
	}
	_,err =hc.tcpConn.Read(n)
		if err!=nil{
			return nil,err
		}
	if req.Header,err = hc.ReadHeader();err!=nil{
		return nil,err
	}
	if req.Header[HeaderHost]==""{
		return nil,&HttpError{"NoHOST",""}
	}
	if req.Method==MethodPost &&req.Header[HeaderContentLength]==""{
		return nil,&HttpError{"NoContentLength",""}
	}
	if req.Header[HeaderContentLength] !=""	{
		length,err:=strconv.Atoi(strings.Replace(req.Header[HeaderContentLength]," ","",-1))
		req.ContentLength=int64(length)
		if err!=nil{
			return nil,&HttpError{"WrongContentLength",req.Header[HeaderContentLength]}
		}
		b:=make([]byte,length)
		n,err:=hc.tcpConn.Read(b)
		if err!=nil{
			return nil,err
		}
		if n!=length{
			return nil,&HttpError{"WrongContentLength",req.Header[HeaderContentLength]}
		}
		req.Body = bytes.NewReader(b)
	}else{
		req.Body=strings.NewReader("")
	}
	return
}
func (hc *httpConn) ReadBytes(delim byte) ([]byte, error){
	var bytes []byte
	b := make([]byte,1)
	for _,err := hc.tcpConn.Read(b); b[0]!=delim; _,err= hc.tcpConn.Read(b){
		if err!=nil {
			return bytes,err
		}
		bytes = append (bytes,b[0])
	}
	return bytes,nil
}
func (hc *httpConn) ReadString(delim byte)(string ,error ){
	bytes, err := hc.ReadBytes(delim)
	return string(bytes),err
}
func parseRequestLine(line string)(method string,URL *url.URL,proto string,ok bool){
	s :=strings.Split(line," ")
	if len(s)!=3{
		return
	}
	Url,err:=url.Parse(s[1])
	if err!=nil{
		return
	}
	return s[0],Url,s[2],true
}
func (hc *httpConn) ReadHeader() (map[string] string , error){
	b:=make([]byte,1)
	_,err:=hc.tcpConn.Read(b)
	if err!=nil{
		return nil,err
	}
	header :=make(map[string]string)
	for b[0] !='\r'{
		str1,err := hc.ReadString(':')
		if err!=nil{
			return nil,err
		}
		str2,err := hc.ReadString('\r')
		if err!=nil{
			return nil,err
		}
		header[string(b)+str1]=str2
		_,err =hc.tcpConn.Read(b);
		if err!=nil{
			return nil,err
		}
		_,err =hc.tcpConn.Read(b);
		if err!=nil{
			return nil,err
		}
	}
		_,err =hc.tcpConn.Read(b);
		if err!=nil{
			return nil,err
		}
	return header,nil
}
