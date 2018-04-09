package http

//
// Http client library.
// Support concurrent and keep-alive (persistent) http requests.
// Not support: chuck transfer encoding.

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"bytes"
	"container/list"
	"sync"
	"sync/atomic"
)

// Client send the http request and recevice response.
//
// Supports concurrency on multiple TCP connections.
type Client struct {
	connSize    int64
	maxConnSize int64

	// Your data here.
	connspool map[string]*list.List
	mu sync.Mutex
	cond sync.Cond
}


// DefaultMaxConnSize is the default max size of
// active tcp connections.
const DefaultMaxConnSize = 500
// NewClient initilize a Client with DefaultMaxConnSize.
func NewClient() *Client {
	return NewClientSize(DefaultMaxConnSize)
}

// NewClientSize initilize a Client with a specific maxConnSize.
func NewClientSize(maxConnSize int64) *Client {
	c := &Client{maxConnSize: maxConnSize}

	// Your initialization code here.
	c.connspool=make(map[string]*list.List)
	c.cond = sync.Cond{L: &c.mu}
	return c
}

// Get implements GET Method of HTTP/1.1.
//
// Must set the body and following headers in the request:
// * Content-Length
// * Host
func (c *Client) Get(URL string) (resp *Response, err error) {
	urlObj, err := url.ParseRequestURI(URL)
	if err != nil {
		return
	}
	header := make(map[string]string)
	header[HeaderContentLength] = "0"
	header[HeaderHost] = urlObj.Host
	req := &Request{
		Method:        MethodGet,
		URL:           urlObj,
		Proto:         HTTPVersion,
		Header:        header,
		ContentLength: 0,
		Body:          strings.NewReader(""),
	}
	resp, err = c.Send(req)
	return
}

// Post implements POST Method of HTTP/1.1.
//
// Must set the body and following headers in the request:
// * Content-Length
// * Host
//
// Write the contentLength bytes data into the body of HTTP request.
// Discard the sequential data after the reading contentLength bytes
// from body(io.Reader).
func (c *Client) Post(URL string, contentLength int64, body io.Reader) (resp *Response, err error) {
	urlObj, err := url.ParseRequestURI(URL)
	if err != nil {
		return
	}
	header := make(map[string]string)
	header[HeaderContentLength] = strconv.FormatInt(contentLength, 10)
	header[HeaderHost] = urlObj.Host
	req := &Request{
		Method:        MethodPost,
		URL:           urlObj,
		Proto:         HTTPVersion,
		Header:        header,
		ContentLength: contentLength,
		Body:          body,
	}
	resp, err = c.Send(req)
	return
}

// Send http request and returns an HTTP response.
//
// An error is returned if caused by client policy (such as invalid
// HTTP response), or failure to speak HTTP (such as a network
// connectivity problem).
//
// Note that a non-2xx status code doesn't mean any above errors.
//
// If the returned error is nil, the Response will contain a non-nil
// Body which is the caller's responsibility to close. If the Body is
// not closed, the Client may not be able to reuse a keep-alive TCP
// connection to the same server.
func (c *Client) Send(req *Request) (resp *Response, err error) {
	if req.URL == nil {
		return nil, errors.New("http: nil Request.URL")
	}

	// Get a available connection to the host for HTTP communication.
	tc, err := c.getConn(req.URL.Host)
	if err != nil {
		return nil, err
	}

	// Write the request to the TCP stream.
	err = c.writeReq(tc, req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.cleanConn(tc)
		return nil, err
	}

	// Construct the response from the TCP stream.
	resp, err = c.constructResp(tc, req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.cleanConn(tc)
	}
	return
}

// Put back the available connection of the specific host
// for the future use.
func (c *Client) putConn(tc *net.TCPConn, host string) {
	// TODO
	c.mu.Lock()
	defer c.mu.Unlock()
	pool, ok := c.connspool[host]
	if ok {
		pool.PushBack(tc)
		c.cond.Broadcast()
	} else {
		panic("not here")
	}
}

// Get a TCP connection to the host.
func (c *Client) getConn(host string) (tc *net.TCPConn, err error) {
	// TODO
	c.mu.Lock()
	pool,ok :=c.connspool[host]
	if !ok{
		pool= list.New()
	        c.connspool[host]=pool
	}
	for pool.Front()==nil && atomic.LoadInt64(&c.connSize) >= c.maxConnSize{
		var maxsize int
		maxsize = 0
		var maxstr  string
		for str,lis :=range c.connspool{
			if str != host{
				if lis.Len()>maxsize{
					maxsize=lis.Len()
					maxstr = str
				}
			}
		}
		if maxsize ==0{
			c.cond.Wait()
		}else{
			tcpconn:=c.connspool[maxstr].Front()
			con:=tcpconn.Value.(*net.TCPConn)
			con.Close()
			c.connspool[maxstr].Remove(tcpconn)
			break
		}
	}
	if pool.Front()==nil{
		atomic.AddInt64(&c.connSize, 1)
		c.mu.Unlock()
		tcpAddr, err := net.ResolveTCPAddr("tcp", host)
		//fmt.Println(tcpAddr)
		if err != nil {
			return nil, err
		}
		tc, err = net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			return nil, err
		}
	}else{
		if ele := pool.Front(); ele != nil {
		tc = ele.Value.(*net.TCPConn)
		pool.Remove(ele)
		c.cond.Broadcast()
		c.mu.Unlock()
		} else {
			panic("not here")
		}
	}
	return
}

// Clean one connection in the case of errors.
func (c *Client) cleanConn(tc *net.TCPConn) {
	// TODO
	tc.Close()
	atomic.AddInt64(&c.connSize, -1)
	c.cond.Broadcast()
}

// Write the request to TCP stream.
//
// The number of bytes in transmit body of a request must be more
// than the value of Content-Length header. If not, throws an error.
func (c *Client) writeReq(tcpConn *net.TCPConn, req *Request) (err error) {
	// TODO
	var bytes []byte
	bytes=[]byte(req.Method+" "+req.URL.String()+" "+req.Proto+"\r\n")
	if _,err=tcpConn.Write(bytes);err!=nil{
		return
	}
	for head,value := range req.Header{
		bytes=[]byte(head+":"+value+"\r\n")
		if _,err=tcpConn.Write(bytes);err!=nil{
			return
		}
	}
	if _,err=tcpConn.Write([]byte("\r\n"));err!=nil{
		return
	}
	if req.ContentLength!=0{
		body :=make([]byte,req.ContentLength)
		if _,err=req.Body.Read(body);err!=nil{
			return err
		}
		if _,err = tcpConn.Write(body);err!=nil{
			return
		}
	}
	return
}

// Construct response from the TCP stream.
//
// Body of the response will return io.EOF if reading
// Content-Length bytes.
//
// If TCP errors occur, err is not nil and req is nil.
func (c *Client) constructResp(tcpConn *net.TCPConn, req *Request) (resp *Response, err error) {
	// TODO
	resp = new(Response)
	resp.Body=new(ResponseReader)
	var respline string
	if respline,err = ReadString(tcpConn,'\r');err!=nil{
		return nil,err
	}
	var ok bool
	n :=make([]byte,1)
	resp.Proto,resp.StatusCode,resp.Status,ok=parseResponseLine(respline)
	if !ok{
		return nil,&HttpError{"WrongResponseScheme",respline}
	}
	if resp.Proto != HTTPVersion{
		return nil,&HttpError{"WrongVersion",resp.Proto}
	}
	_,err=tcpConn.Read(n)
	if err!=nil{
		return nil,err
	}
	if resp.Header,err =ReadHeader(tcpConn);err!=nil{
		return nil,err
	}
	if resp.Header[HeaderContentLength]==""{
		return nil,&HttpError{"NoContentLength",""}
	}
	length,err:=strconv.Atoi(strings.Replace(resp.Header[HeaderContentLength]," ","",-1))
	resp.ContentLength=int64(length)
		if err!=nil{
			return nil,&HttpError{"WrongContentLength",resp.Header[HeaderContentLength]}
		}
	if length!=0{
		b:=make([]byte,length)
		num,err:=tcpConn.Read(b)
		if err!=nil{
			return nil,err
		}
		if num!=length{
		return nil,&HttpError{"WrongContentLength",resp.Header[HeaderContentLength]}
		}
		resp.Body.r = bytes.NewReader(b)
	}else{
		resp.Body.r = strings.NewReader("")
	}
	resp.Body.tc = tcpConn
	resp.Body.c = c
	resp.Body.host = req.URL.Host
	return
}
func  ReadBytes(tc *net.TCPConn,delim byte) ([]byte, error){
	var bytes []byte
	b := make([]byte,1)
	for _,err := tc.Read(b); b[0]!=delim; _,err= tc.Read(b){
		if err!=nil {
			return bytes,err
		}
		bytes = append (bytes,b[0])
	}
	return bytes,nil
}
func  ReadString(tc *net.TCPConn,delim byte)(string ,error ){
	bytes, err :=ReadBytes(tc,delim)
	return string(bytes),err
}
func parseResponseLine(line string) (Proto string,StatusCode int, Statue string, ok bool){
	s :=strings.SplitN(line," ",3)
	code,err:=strconv.Atoi(s[1])
	if err!=nil{
		return
	}
	return s[0],code,s[2],true
}
func ReadHeader(tc *net.TCPConn)(map[string]string,error){
	b:=make([]byte,1)
	_,err:=tc.Read(b)
	if err!=nil{
		return nil,err
	}
	header :=make(map[string]string)
	for b[0] !='\r'{
		str1,err := ReadString(tc,':')
		if err!=nil{
			return nil,err
		}
		str2,err := ReadString(tc,'\r')
		if err!=nil{
			return nil,err
		}
		header[string(b)+str1]=str2
		_,err =tc.Read(b);
		if err!=nil{
			return nil,err
		}
		_,err =tc.Read(b);
		if err!=nil{
			return nil,err
		}
	}
		_,err =tc.Read(b);
		if err!=nil{
			return nil,err
		}
	return header,nil
}
