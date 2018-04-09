package http

import(
	"testing"
	go_http "net/http"
	"time"
	"bytes"
	"io/ioutil"
)

//Server host 
const(
	ServerAddr_1= "localhost:9000"
	HTTPHost_1   = "http://" + ServerAddr_1
	ServerAddr_2 = "localhost:9001"
	HTTPHost_2   = "http://" + ServerAddr_2
)
func YourServer(host string) (server *Server, ch chan error) {
	ch = make(chan error)
	server = NewServer(host)

	// start server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != ErrServerClosed {
			ch <- err
		} else {
			close(ch)
		}
	}()
	time.Sleep(time.Second)
	return server, ch
}

func GoServer(host string) (server *go_http.Server, serverMux *go_http.ServeMux, ch chan error) {
	ch = make(chan error)
	serverMux = go_http.NewServeMux()
	server = &go_http.Server{
		Addr:    host,
		Handler: serverMux,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != go_http.ErrServerClosed {
			ch <- err
		} else {
			close(ch)
		}
	}()
	time.Sleep(time.Second)
	return server, serverMux, ch
}

func checkClient(t *testing.T, client *Client, host string,path string, method string,
	reqBodyData []byte, statusCode int, expectedRespBodyData []byte) {
	url := host + path
	var resp *Response
	var err error
	if method == MethodGet {
		resp, err = client.Get(url)
	} else {
		resp, err = client.Post(url, int64(len(reqBodyData)), bytes.NewReader(reqBodyData))
	}
	if err != nil || resp == nil {
		t.Fatalf("Get(%v) failed, error: %v", url, err)
	} else {
		if resp.StatusCode != statusCode {
			t.Fatalf("Get(%v) status=%v, expected=%v", url, resp.StatusCode, statusCode)
		}
		respBodyData, _ := ioutil.ReadAll(resp.Body)
		if bytes.Compare(respBodyData, expectedRespBodyData) != 0 {
			t.Fatalf("Get(%v) body=%v, expected=%v", url, string(respBodyData), string(expectedRespBodyData))
		}
	}
}
func GocheckClient(t *testing.T, client *go_http.Client, host string,path string, method string,
	reqBodyData []byte, statusCode int, expectedRespBodyData []byte){
		url := host + path
		var resp *go_http.Response
		var err error
		if method == MethodGet {
			resp, err = client.Get(url)
		} else {
			resp, err = client.Post(url, HeaderContentTypeValue, bytes.NewReader(reqBodyData))
		}
		if err != nil || resp == nil {
			t.Fatalf("Get(%v) failed, error: %v", url, err)
		} else {
			if resp.StatusCode != statusCode {
				t.Fatalf("Get(%v) status=%v, expected=%v",
					url, resp.StatusCode, statusCode)
			}
			respBodyData, _ := ioutil.ReadAll(resp.Body)
			if bytes.Compare(respBodyData, expectedRespBodyData) != 0 {
				t.Fatalf("Get(%v) body=%v, expected=%v",
					url, string(respBodyData), string(expectedRespBodyData))
			}
		}
}

	
