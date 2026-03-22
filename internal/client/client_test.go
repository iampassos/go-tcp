package client

import (
	"testing"
)

func TestHandshake(t *testing.T) {
	client := NewClient()

	t.Run("client sends SYN and changes state", func(t *testing.T) {
		err := client.StartHandshake()
		if err != nil {
			t.Fatal(err)
		}

		if client.State != SYN_SENT {
			t.Fatalf("got state %d, wanted %d", client.State, SYN_SENT)
		}
	})
}
