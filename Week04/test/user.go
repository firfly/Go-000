package main

import (
	"context"
	"log"
	"time"

	v1 "Week04/api/user/v1"

	"google.golang.org/grpc"
)

const (
	address = "localhost:9800"
	name    = "zhangsan"
	age     = 20
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// grpc client
	c := v1.NewUserClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// call rpc
	r, err := c.RegisterUser(ctx, &v1.RegisterUserRequest{Name: name, Age: age})
	if err != nil {
		log.Fatalf("register user failed: %v", err)
	}
	log.Printf("register user id: %d", r.GetId())
}