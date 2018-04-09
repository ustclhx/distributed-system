package http
import(
	"fmt"
	"testing"
	go_http "net/http"
	"strconv"
)

func TestDecoupled(t *testing.T){
	your_server,your_sCloseChan := YourServer(ServerAddr_1)
	go_server, go_serverMux, go_sCloseChan := GoServer(ServerAddr_2)
	var value int64
	your_server.AddHandlerFunc("/add", wrapYourAddFunc(&value))
	your_server.AddHandlerFunc("/value", wrapYourValueFunc(&value))
	go_serverMux.HandleFunc("/add", wrapGoAddFunc(&value))
	go_serverMux.HandleFunc("/value", wrapGoValueFunc(&value))
	your_client := NewClient()
	go_client:= new(go_http.Client)
	fmt.Printf("Test:decoupled...\n")
	
	GocheckClient(t, go_client,HTTPHost_1, "/add", MethodPost, []byte("10"), StatusOK, []byte(""))
	if value != 10 {
		t.Fatalf("value -> %v, expected %v", value, 10)
	}
	GocheckClient(t,go_client,HTTPHost_1, "/value", MethodGet, []byte{}, StatusOK, []byte(strconv.Itoa(int(value))))

	GocheckClient(t,go_client,HTTPHost_1, "/add", MethodPost, []byte("-5"), StatusOK, []byte(""))
	if value != 5 {
		t.Fatalf("value -> %v, expected %v", value, 5)
	}
	GocheckClient(t, go_client,HTTPHost_1, "/value", MethodGet, []byte{}, StatusOK, []byte(strconv.Itoa(int(value))))
	fmt.Println("go client to your server passed")

	GocheckClient(t, go_client,HTTPHost_2, "/add", MethodPost, []byte("5"), StatusOK, []byte(""))
	if value != 10 {
		t.Fatalf("value -> %v, expected %v", value, 10)
	}
	GocheckClient(t,go_client,HTTPHost_2, "/value", MethodGet, []byte{}, StatusOK, []byte(strconv.Itoa(int(value))))

	GocheckClient(t,go_client,HTTPHost_2, "/add", MethodPost, []byte("-10"), StatusOK, []byte(""))
	if value != 0 {
		t.Fatalf("value -> %v, expected %v", value, 05)
	}
	GocheckClient(t, go_client,HTTPHost_2, "/value", MethodGet, []byte{}, StatusOK, []byte(strconv.Itoa(int(value))))
	fmt.Println("go client to go server passed")

	checkClient(t, your_client,HTTPHost_1, "/add", MethodPost, []byte("10"), StatusOK, []byte(""))
	if value != 10 {
		t.Fatalf("value -> %v, expected %v", value, 10)
	}
	checkClient(t,your_client,HTTPHost_1, "/value", MethodGet, []byte{}, StatusOK, []byte(strconv.Itoa(int(value))))

	checkClient(t,your_client,HTTPHost_1, "/add", MethodPost, []byte("-5"), StatusOK, []byte(""))
	if value != 5 {
		t.Fatalf("value -> %v, expected %v", value, 5)
	}
	checkClient(t, your_client,HTTPHost_1, "/value", MethodGet, []byte{}, StatusOK, []byte(strconv.Itoa(int(value))))
	fmt.Println("your client to your server passed")

	checkClient(t, your_client,HTTPHost_2, "/add", MethodPost, []byte("5"), StatusOK, []byte(""))
	if value != 10 {
		t.Fatalf("value -> %v, expected %v", value, 10)
	}
	checkClient(t,your_client,HTTPHost_2, "/value", MethodGet, []byte{}, StatusOK, []byte(strconv.Itoa(int(value))))

	checkClient(t,your_client,HTTPHost_2, "/add", MethodPost, []byte("-10"), StatusOK, []byte(""))
	if value != 0 {
		t.Fatalf("value -> %v, expected %v", value, 05)
	}
	checkClient(t, your_client,HTTPHost_2, "/value", MethodGet, []byte{}, StatusOK, []byte(strconv.Itoa(int(value))))
	fmt.Println("your client to go server passed")

	your_server.Close()
	if err := <-your_sCloseChan; err == nil {
		fmt.Printf("your Server closed\n")
	} else {
		t.Fatalf("%v", err)
	}
	go_server.Close()
	if err := <-go_sCloseChan; err == nil {
		fmt.Printf("go Server closed\n")
	} else {
		t.Fatalf("%v", err)
	}
	fmt.Printf("  ... Passed\n")
}
