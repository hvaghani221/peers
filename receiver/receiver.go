package main

import (
	"fmt"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/kademlia"
)

func main() {
	me, err := noise.NewNode()
	if err != nil {
		panic(err)
	}
	defer me.Close()

	protocol := kademlia.New()
	me.Bind(protocol.Protocol())

	wait := make(chan struct{})

	me.Handle(func(ctx noise.HandlerContext) error {
		fmt.Printf("Got a message from %s\ndata: '%s'\n\n", ctx.ID().String(), string(ctx.Data()))
		close(wait)
		return nil
	})

	if err := me.Listen(); err != nil {
		panic(err)
	}
	fmt.Println("My Address: ", me.Addr())

	for range wait {
	}
}
