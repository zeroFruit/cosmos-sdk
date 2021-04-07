package client

import (
	"context"
	"testing"
	"time"
)

func TestDial(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c, err := Dial(ctx, "", "localhost:9090")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", c)
}
