package client

import (
	"testing"

	"github.com/iampassos/go-tcp/internal/server"
)

func TestHandshake(t *testing.T) {
	sv := server.NewServer()
	go sv.Start()

	client := NewClient()

	err := client.Init()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("client sends SYN and changes state", func(t *testing.T) {
		err := client.InitHandshake()
		if err != nil {
			t.Fatal(err)
		}

		if client.State != SYN_SENT {
			t.Fatalf("got state %d, wanted %d", client.State, SYN_SENT)
		}
	})
}
