package http

import(
	"testing"
	"time"
	"bytes"
	"fmt"
	"strconv"
)

//test two distant connections to different servers
func TestTwoserver(t *testing.T){
	server_1, sCloseChan_1 :=YourServer(ServerAddr_1)
	server_2, sCloseChan_2 := YourServer(ServerAddr_2)
	var value int64
	server_1.AddHandlerFunc("/add", wrapYourAddFunc(&value))
	server_1.AddHandlerFunc("/value", wrapYourValueFunc(&value))
	server_2.AddHandlerFunc("/add", wrapYourAddFunc(&value))
	server_2.AddHandlerFunc("/value", wrapYourValueFunc(&value))
	c:=NewClientSize(5)
	fmt.Printf("Test: Connect to two different server...\n")
	incrReqNum, decrReqNum := 10, 20
	type Status struct {
		flag bool
		url  string
		err  error
	}	
	waitComplete_1 := make(chan Status, incrReqNum+decrReqNum)
	waitComplete_2 := make(chan Status, incrReqNum+decrReqNum)
	go func() {
		for i := 0; i < incrReqNum; i++ {
			go func(ii int) {
				reqBodyData := []byte("1")
				if resp, err := c.Post(HTTPHost_1+"/add", int64(len(reqBodyData)),
					bytes.NewReader(reqBodyData)); err != nil || resp == nil {
					waitComplete_1 <- Status{flag: false, url: "/add", err: err}
				} else {
					resp.Body.Close()
					waitComplete_1 <- Status{flag: true}
				}
			}(i)
		}
	}()

	go func() {
		for i := 0; i < decrReqNum; i++ {
			go func(ii int) {
				reqBodyData := []byte("-1")
				if resp, err := c.Post(HTTPHost_1+"/add", int64(len(reqBodyData)),
					bytes.NewReader(reqBodyData)); err != nil || resp == nil {
					waitComplete_1 <- Status{flag: false, url: "/add", err: err}
				} else {
					resp.Body.Close()
					waitComplete_1 <- Status{flag: true}
				}
			}(i)
		}
	}()
	go func() {
		time.Sleep(1*time.Second)
		for i := 0; i < incrReqNum; i++ {
			go func(ii int) {
				reqBodyData := []byte("1")
				if resp, err := c.Post(HTTPHost_2+"/add", int64(len(reqBodyData)),
					bytes.NewReader(reqBodyData)); err != nil || resp == nil {
					waitComplete_2 <- Status{flag: false, url: "/add", err: err}
				} else {
					resp.Body.Close()
					waitComplete_2 <- Status{flag: true}
				}
			}(i)
		}
	}()

	go func() {
		time.Sleep(1*time.Second)
		for i := 0; i < decrReqNum; i++ {
			go func(ii int) {
				reqBodyData := []byte("-1")
				if resp, err := c.Post(HTTPHost_2+"/add", int64(len(reqBodyData)),
					bytes.NewReader(reqBodyData)); err != nil || resp == nil {
					waitComplete_2 <- Status{flag: false, url: "/add", err: err}
				} else {
					resp.Body.Close()
					waitComplete_2 <- Status{flag: true}
				}
			}(i)
		}
	}()
	for i := 0; i < cap(waitComplete_1); i++ {
		select {
		case status := <-waitComplete_1:
			if !status.flag {
				t.Fatalf("Get(%v) failed, error:%v", status.url, status.err)
			}
		case <-time.After(time.Millisecond * 500):
			t.Fatalf("wait reply timeout")
		}
	}
	time.Sleep(1*time.Second)
	for i := 0; i < cap(waitComplete_2); i++ {
		select {
		case status := <-waitComplete_2:
			if !status.flag {
				t.Fatalf("Get(%v) failed, error:%v", status.url, status.err)
			}
		case <-time.After(time.Millisecond * 500):
			t.Fatalf("wait reply timeout")
		}
	}

	expectedValue := int64(2*(incrReqNum - decrReqNum))
	if value != expectedValue {
		t.Fatalf("value=%v, expected=%v", value, expectedValue)
	}
	checkClient(t, c, HTTPHost_1, "/value", MethodGet, []byte{}, StatusOK, []byte(strconv.Itoa(int(value))))
	checkClient(t, c, HTTPHost_2, "/value", MethodGet, []byte{}, StatusOK, []byte(strconv.Itoa(int(value))))

	server_1.Close()
	if err := <-sCloseChan_1; err == nil {
		fmt.Printf("Server1 closed\n")
	} else {
		t.Fatalf("%v", err)
	}
	server_2.Close()
	if err := <-sCloseChan_2; err == nil {
		fmt.Printf("Server2 closed\n")
	} else {
		t.Fatalf("%v", err)
	}
	fmt.Printf("  ... Passed\n")
}
