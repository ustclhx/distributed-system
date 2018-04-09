package http

import(
	"fmt"
	"flag"
	"time"
	"strconv"
	"bytes"
	"testing"
)

func TestPerformance(t *testing.T){
	var servernumber,clientnumber,connsnumber int
	flag.IntVar(&servernumber,"servernumber",1,"set the number of servers")
	flag.IntVar(&clientnumber,"clientnumber",1,"set the number of clients")
	flag.IntVar(&connsnumber,"connsnumber",100,"set the number of connections between each client and server")
	flag.Parse()
	fmt.Printf("servernumer:%v,clientnumber:%v,per connections:%v\n",servernumber,clientnumber,connsnumber)
	serveraddrs:= make([]string,servernumber)
	HTTPHosts:=make([]string,servernumber)
	for i:=0;i<servernumber;i++{
		serveraddrs[i]="localhost:"+strconv.Itoa(9000+i)
		HTTPHosts[i]="http://"+serveraddrs[i]
	}
	var value int64 
	servers := make([]*Server,servernumber)
	sCloseChans :=make([]chan error,servernumber)
	for i:=0;i<servernumber;i++{
		servers[i],sCloseChans[i]=YourServer(serveraddrs[i])
		servers[i].AddHandlerFunc("/add", wrapYourAddFunc(&value))
		servers[i].AddHandlerFunc("/value", wrapYourValueFunc(&value))
	}
	clients := make([]*Client,clientnumber)
	for i:=0;i<clientnumber;i++{
		clients[i]=NewClientSize(50)
	}
	type Status struct{
		flag bool
		url string
		err error
	}
	waitComplete :=make(chan Status,servernumber*clientnumber*connsnumber)
	start:=time.Now()
	for i:=0;i<servernumber;i++{
		for j:=0;j<clientnumber;j++{
			m:=i
			n:=j
			for count:=0;count<connsnumber;count++{
				go func(ii int) {
					reqBodyData := []byte("1")
					if resp, err := clients[n].Post(HTTPHosts[m]+"/add", int64(len(reqBodyData)),
						bytes.NewReader(reqBodyData)); err != nil || resp == nil {
						waitComplete <- Status{flag: false, url: "/add", err: err}
					} else {
						resp.Body.Close()
						waitComplete <- Status{flag: true}
					}
				}(count)
			}
		}
	}
	for i := 0; i < cap(waitComplete); i++ {
		select {
		case status := <-waitComplete:
			if !status.flag {
				t.Fatalf("Get(%v) failed, error:%v", status.url, status.err)
			}
		case <-time.After(time.Millisecond * 500):
			t.Fatalf("wait reply timeout")
		}
	}
	elapsedMs := time.Since(start).Nanoseconds() / int64(time.Millisecond)
	fmt.Printf("Cost: %v ms\n", elapsedMs)	
	for i:=0;i<servernumber;i++{
		servers[i].Close()
		if err := <-sCloseChans[i]; err == nil {
			fmt.Printf("Server%v closed\n",i)
		} else {
			t.Fatalf("%v", err)
		}
	}
}




