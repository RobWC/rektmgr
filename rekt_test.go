package rektmgr

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestNewREKTManager(t *testing.T) {
	t.Log("Starting REKT Manager")
	rm := NewREKTManager(1)
	if rm == nil {
		t.Fatal("REKT Manager not created")
	}
	t.Log("Closing REKT Manager")
	rm.Close()
}

func TestNewREKTManagerRequest(t *testing.T) {
	t.Log("Starting REKT Manager")
	rm := NewREKTManager(1)
	rm.SetRespHandler(func(data []byte, err error) {
		if err != nil {
			t.Log("Error received:", err)
		}
		t.Log("Data received:", len(data))
	})

	t.Log("Send request")
	req, err := http.NewRequest("GET", "http://localhost:6060", nil)
	if err != nil {
		t.Fatal(err)
	}
	rm.Do(*req)
	timer := time.NewTimer(time.Second * 5)
	<-timer.C
	rm.Close()
	timer.Stop()
}

func TestNewREKTManagerMultiRequest(t *testing.T) {
	workers := 100
	fmt.Println("Starting REKT Manager")
	rm := NewREKTManager(workers)
	fmt.Println("Setting resp handler")

	counter := 0
	rm.SetRespHandler(func(data []byte, err error) {
		if err != nil {
			t.Log("Error received:", err)
		}
		t.Log("Data received:", len(data))
		t.Logf("Task %d complete", counter)
		counter++
	})

	fmt.Println("Starting jobs")
	for i := 0; workers > i; i++ {
		fmt.Println("Send request", i)
		req, err := http.NewRequest("GET", "http://localhost:6060/doc/", nil)
		if err != nil {
			t.Fatal(err)
		}
		rm.Do(*req)
	}
	fmt.Println("Jobs sent")
	rm.Close()
}
