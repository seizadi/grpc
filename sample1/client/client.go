package main

import (
	"fmt"
	"google.golang.org/grpc"
	context "golang.org/x/net/context"
	"../proto"
	"google.golang.org/grpc/metadata"
	"os"
)

func main() {
	// First we create the connection:
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	
	// We can now create stubs that wrap conn:
	stub := proto.NewStarfriendsClient(conn)
	
	// Now we can use the stub to make RPCs
	// Now we can use the stub to make RPCs
	ctx := metadata.NewOutgoingContext(context.Background(),
		metadata.Pairs("who", "starfiends-go-client", "version", "v1"))
	req := &proto.GetFilmRequest{Id: "4"}
	resp, err := stub.GetFilm(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RPC failed: %v\n", err)
	} else {
		fmt.Println(resp)
	}
	
	// We'll make another request and also print the response metadata
	req = &proto.GetFilmRequest{Id: "7"}
	var respHdrs, respTrlrs metadata.MD
	resp, err = stub.GetFilm(ctx, req,
		grpc.Header(&respHdrs), grpc.Trailer(&respTrlrs))
	if err != nil {
		fmt.Fprintf(os.Stderr, "RPC failed: %v\n", err)
	} else {
		fmt.Println(resp)
	}
	fmt.Printf("Server sent headers: %v\n", respHdrs)
	fmt.Printf("Server sent trailers: %v\n", respTrlrs)
}
