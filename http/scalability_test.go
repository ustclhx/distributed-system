package http 

import(
	"fmt"
	"testing"
	"time"
	"bytes"
	"strconv"
)
//keep the load fixed , increase the number of servers
func TestScalability_1(t *testing.T){
	servernumber:=7
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
	c := NewClientSize(100)
	num:=5040
	type Status struct{
		flag bool
		url string
		err error
	}
	waitComplete :=make(chan Status,num)
	//index:=make(chan int, num)
	for i:=1;i<=servernumber;i++{
		n:=i
		fmt.Printf("connect to  %v server:\n",n)
		/*for n:=0;n<num/i;n++{
			for m:=0;m<i;m++{
				index<-m
			}
		}*/
		start:=time.Now()
		for j:=0;j<n;j++{
			m:=j
			go func(){
				for jj:=0;jj<num/n;jj++{
					go func(ii int){
						reqBodyData := []byte("1")
						if resp, err := c.Post(HTTPHosts[m]+"/add", int64(len(reqBodyData)),
							bytes.NewReader(reqBodyData)); err != nil || resp == nil {
							waitComplete <- Status{flag: false, url: "/add", err: err}
						} else {
							resp.Body.Close()
							waitComplete <- Status{flag: true}
						}
					}(jj)
				}
			}()
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
	}
	for i:=0;i<servernumber;i++{
		servers[i].Close()
		if err := <-sCloseChans[i]; err == nil {
			fmt.Printf("Server%v closed\n",i)
		} else {
			t.Fatalf("%v", err)
		}
	}
}

//keep the load fixed,increase the number of clients
func TestScalability_2(t *testing.T){
	serveraddr:="localhost:9000"
	HTTPhost:="http://"+serveraddr
	var value int64
	clientnumber:=7
	server,sCloseChan:=YourServer(serveraddr)
	server.AddHandlerFunc("/add", wrapYourAddFunc(&value))
	server.AddHandlerFunc("/value", wrapYourValueFunc(&value))
	clients := make([]*Client,clientnumber)
	for i:=0;i<clientnumber;i++{
		clients[i]=NewClientSize(40)
	}
	num:=5040
	type Status struct{
		flag bool
		url string
		err error
	}
	waitComplete :=make(chan Status,num)
	//index:=make(chan int,num)
	for i:=1;i<=clientnumber;i++{
		n:=i
		fmt.Printf(" %v client dial connectionns:\n",n)
		/*for n:=0;n<num/i;n++{
			for m:=0;m<i;m++{
				index<-m
			}
		}*/
		start:=time.Now()
		for j:=0;j<n;j++{
			m:=j
			go func(){
				for jj:=0;jj<num/n;jj++{
					go func(ii int){
						reqBodyData := []byte("1")
						if resp, err := clients[m].Post(HTTPhost+"/add", int64(len(reqBodyData)),
							bytes.NewReader(reqBodyData)); err != nil || resp == nil {
							waitComplete <- Status{flag: false, url: "/add", err: err}
						} else {
							resp.Body.Close()
							waitComplete <- Status{flag: true}
						}
					}(jj)
				}
			}()
		}
	
		for j := 0; j < cap(waitComplete); j++ {
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
		for j:=0;j<i;j++{
			clients[j]=NewClientSize(40)
		}
	}
	server.Close()
		if err := <-sCloseChan; err == nil {
			fmt.Printf("Server closed\n")
		} else {
			t.Fatalf("%v", err)
		}
}
//keep the load fixed,increase the number of clients and servers(1-1)
func TestScalability_3(t *testing.T){
	nodenumber:=7
	serveraddrs:= make([]string,nodenumber)
	HTTPHosts:=make([]string,nodenumber)
	for i:=0;i<nodenumber;i++{
		serveraddrs[i]="localhost:"+strconv.Itoa(9000+i)
		HTTPHosts[i]="http://"+serveraddrs[i]
	}
	values :=make([]int64,nodenumber)
	servers := make([]*Server,nodenumber)
	sCloseChans :=make([]chan error,nodenumber)
	for i:=0;i<nodenumber;i++{
		servers[i],sCloseChans[i]=YourServer(serveraddrs[i])
		servers[i].AddHandlerFunc("/add", wrapYourAddFunc(&values[i]))
		servers[i].AddHandlerFunc("/value", wrapYourValueFunc(&values[i]))
	}
	clients := make([]*Client,nodenumber)
	for i:=0;i<nodenumber;i++{
		clients[i]=NewClientSize(40)
	}
	num:=5040
	type Status struct{
		flag bool
		url string
		err error
	}
	waitComplete :=make(chan Status,num)
	//index:=make(chan int,num)
	for i:=1;i<=nodenumber;i++{
		n:=i
		fmt.Printf(" %v client dial connectionns:\n",n)
		/*for n:=0;n<num/i;n++{
			for m:=0;m<i;m++{
				index<-m
			}
		}*/
		start:=time.Now()
		for j:=0;j<n;j++{
			m:=j
			go func(){
				for jj:=0;jj<num/n;jj++{
					go func(ii int){
						//in:=<-index
						reqBodyData := []byte("1")
						if resp, err := clients[m].Post(HTTPHosts[m]+"/add", int64(len(reqBodyData)),
							bytes.NewReader(reqBodyData)); err != nil || resp == nil {
							waitComplete <- Status{flag: false, url: "/add", err: err}
						} else {
							resp.Body.Close()
							waitComplete <- Status{flag: true}
						}
					}(jj)
				}
			}()
		}
	
		for j := 0; j < cap(waitComplete); j++ {
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
		for j:=0;j<i;j++{
			clients[j]=NewClientSize(40)
		}
	}
	for i:=0;i<nodenumber;i++{
		servers[i].Close()
		if err := <-sCloseChans[i]; err == nil {
			fmt.Printf("Server%v closed\n",i)
		} else {
			t.Fatalf("%v", err)
		}
	}
}
//keep the load fixed,increase the number of clients and servers(n-n)
func TestScalability_4(t *testing.T){
	nodenumber:=6
	serveraddrs:= make([]string,nodenumber)
	HTTPHosts:=make([]string,nodenumber)
	for i:=0;i<nodenumber;i++{
		serveraddrs[i]="localhost:"+strconv.Itoa(9000+i)
		HTTPHosts[i]="http://"+serveraddrs[i]
	}
	values :=make([]int64,nodenumber)
	servers := make([]*Server,nodenumber)
	sCloseChans :=make([]chan error,nodenumber)
	for i:=0;i<nodenumber;i++{
		servers[i],sCloseChans[i]=YourServer(serveraddrs[i])
		servers[i].AddHandlerFunc("/add", wrapYourAddFunc(&values[i]))
		servers[i].AddHandlerFunc("/value", wrapYourValueFunc(&values[i]))
	}
	clients := make([]*Client,nodenumber)
	for i:=0;i<nodenumber;i++{
		clients[i]=NewClientSize(40)
	}
	num:=3600
	type Status struct{
		flag bool
		url string
		err error
	}
	waitComplete :=make(chan Status,num)
	//index:=make(chan int,num)
	for i:=1;i<=nodenumber;i++{
		n:=i
		fmt.Printf(" %v client dial connectionns:\n",n)
		/*for n:=0;n<num/(i*i);n++{
			for m:=0;m<i;m++{
				index<-m
			}
		}*/
		start:=time.Now()
		for j:=0;j<n;j++{
			m:=j
			go func(){
				for jj:=0;jj<num/(n*n);jj++{
					go func(ii int){
						//in:=<-index
						reqBodyData := []byte("1")
						for k:=0;k<n;k++{
							if resp, err := clients[m].Post(HTTPHosts[k]+"/add", int64(len(reqBodyData)),
								bytes.NewReader(reqBodyData)); err != nil || resp == nil {
								waitComplete <- Status{flag: false, url: "/add", err: err}
							} else {
								resp.Body.Close()
								waitComplete <- Status{flag: true}
							}
						}
					}(jj)
				}
			}()
		}	
		for j := 0; j < cap(waitComplete); j++ {
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
		for j:=0;j<i;j++{
			clients[j]=NewClientSize(40)
		}
	}
	for i:=0;i<nodenumber;i++{
		servers[i].Close()
		if err := <-sCloseChans[i]; err == nil {
			fmt.Printf("Server%v closed\n",i)
		} else {
			t.Fatalf("%v", err)
		}
	}
}
//keep load/node fixed, increase them together(1-1)
func TestScalability_5(t *testing.T){
	nodenumber:=10
	serveraddrs:= make([]string,nodenumber)
	HTTPHosts:=make([]string,nodenumber)
	for i:=0;i<nodenumber;i++{
		serveraddrs[i]="localhost:"+strconv.Itoa(9000+i)
		HTTPHosts[i]="http://"+serveraddrs[i]
	}
	values :=make([]int64,nodenumber)
	servers := make([]*Server,nodenumber)
	sCloseChans :=make([]chan error,nodenumber)
	for i:=0;i<nodenumber;i++{
		servers[i],sCloseChans[i]=YourServer(serveraddrs[i])
		servers[i].AddHandlerFunc("/add", wrapYourAddFunc(&values[i]))
		servers[i].AddHandlerFunc("/value", wrapYourValueFunc(&values[i]))
	}
	clients := make([]*Client,nodenumber)
	for i:=0;i<nodenumber;i++{
		clients[i]=NewClientSize(40)
	}
	connnum:=1000
	type Status struct{
		flag bool
		url string
		err error
		index int
	}
	for i:=1;i<nodenumber;i++{
		n:=i
		waitComplete:=make(chan Status,connnum*n) 
		//index:=make(chan int,connnum*i)
		count:=make([]int,n)
		fmt.Printf(" %v client dial connectionns:\n",n)
		/*for n:=0;n<connnum;n++{
			for m:=0;m<i;m++{
				index<-m
			}
		}*/
		start:=time.Now()
		for j:=0;j<n;j++{
			m:=j
			go func(){
				for jj:=0;jj<connnum;jj++{
					go func(ii int){
						//in:=<-index
						reqBodyData := []byte("1")
						if resp, err := clients[m].Post(HTTPHosts[m]+"/add", int64(len(reqBodyData)),
							bytes.NewReader(reqBodyData)); err != nil || resp == nil {
							waitComplete <- Status{flag: false, url: "/add", err: err,index:m}
						} else {
							resp.Body.Close()
							waitComplete <- Status{flag: true,index:m}
						}
					}(jj)
				}
			}()
		}
		for n := 0; n < cap(waitComplete); n++ {
			select {
				case status := <-waitComplete:
					if !status.flag {
						t.Fatalf("Get(%v) failed, error:%v", status.url, status.err)
					}else{
						count[status.index]++
						if(count[status.index]>=connnum){
							elapsedMs := time.Since(start).Nanoseconds() / int64(time.Millisecond)
							fmt.Printf("client %v Cost: %v ms\n", status.index,elapsedMs)
						}	
					}
					case <-time.After(time.Millisecond * 500):
						t.Fatalf("wait reply timeout")
				}
		}
		for j:=0;j<i;j++{
			clients[j]=NewClientSize(40)
		}
	
	}
	for i:=0;i<nodenumber;i++{
		servers[i].Close()
		if err := <-sCloseChans[i]; err == nil {
			fmt.Printf("Server%v closed\n",i)
		} else {
			t.Fatalf("%v", err)
		}
	}
}
//keep load/node fixed, increase them together(n-n)
func TestScalability_6(t *testing.T){
	nodenumber:=6
	serveraddrs:= make([]string,nodenumber)
	HTTPHosts:=make([]string,nodenumber)
	for i:=0;i<nodenumber;i++{
		serveraddrs[i]="localhost:"+strconv.Itoa(9000+i)
		HTTPHosts[i]="http://"+serveraddrs[i]
	}
	values :=make([]int64,nodenumber)
	servers := make([]*Server,nodenumber)
	sCloseChans :=make([]chan error,nodenumber)
	for i:=0;i<nodenumber;i++{
		servers[i],sCloseChans[i]=YourServer(serveraddrs[i])
		servers[i].AddHandlerFunc("/add", wrapYourAddFunc(&values[i]))
		servers[i].AddHandlerFunc("/value", wrapYourValueFunc(&values[i]))
	}
	clients := make([]*Client,nodenumber)
	for i:=0;i<nodenumber;i++{
		clients[i]=NewClientSize(40)
	}
	connnum:=720
	type Status struct{
		flag bool
		url string
		err error
		index int
	}
	for i:=1;i<nodenumber;i++{
		n:=i
		waitComplete:=make(chan Status,connnum*n) 
		//index:=make(chan int,connnum*i)
		count:=make([]int,n)
		fmt.Printf(" %v client dial connectionns:\n",n)
		/*for n:=0;n<connnum/i;n++{
			for m:=0;m<i;m++{
				index<-m
			}
		}*/
		start:=time.Now()
		for j:=0;j<n;j++{
			m:=j
			go func(){
				for jj:=0;jj<connnum/n;jj++{
					go func(ii int){
						//in:=<-index
						reqBodyData := []byte("1")
						for n:=0;n<i;n++{
							//fmt.Println(in,n)
							if resp, err := clients[m].Post(HTTPHosts[n]+"/add", int64(len(reqBodyData)),
								bytes.NewReader(reqBodyData)); err != nil || resp == nil {
								waitComplete <- Status{flag: false, url: "/add", err: err,index:m}
							} else {
								resp.Body.Close()
								waitComplete <- Status{flag: true,index:m}
							}
						}
					}(jj)
				}
			}()
		}
		for n := 0; n < cap(waitComplete); n++ {
			select {
				case status := <-waitComplete:
					if !status.flag {
						t.Fatalf("Get(%v) failed, error:%v", status.url, status.err)
					}else{
						count[status.index]++
						if(count[status.index]>=connnum){
							elapsedMs := time.Since(start).Nanoseconds() / int64(time.Millisecond)
							fmt.Printf("client %v Cost: %v ms\n", status.index,elapsedMs)
						}	
					}
					case <-time.After(time.Millisecond * 500):
						t.Fatalf("wait reply timeout")
				}
		}
		for j:=0;j<i;j++{
			clients[j]=NewClientSize(40)
		}
	
	}
	for i:=0;i<nodenumber;i++{
		servers[i].Close()
		if err := <-sCloseChans[i]; err == nil {
			fmt.Printf("Server%v closed\n",i)
		} else {
			t.Fatalf("%v", err)
		}
	}
}




