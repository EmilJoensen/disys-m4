package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	ping "github.com/EmilJoensen/disys-m4/grpc"
	"google.golang.org/grpc"
)

func main() {
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	ownPort := int32(arg1) + 8000

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &peer{
		id:            ownPort,
		amountOfPings: make(map[int32]int32),
		clients:       make(map[int32]ping.PingClient),
		ctx:           ctx,
	}

	// Create listener tcp on port ownPort
	list, err := net.Listen("tcp", fmt.Sprintf(":%v", ownPort))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}
	grpcServer := grpc.NewServer()
	ping.RegisterPingServer(grpcServer, p)

	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to server %v", err)
		}
	}()

	for i := 0; i < 3; i++ {
		port := int32(8000) + int32(i)

		if port == ownPort {
			continue
		}

		var conn *grpc.ClientConn
		fmt.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("Could not connect: %s", err)
		}
		defer conn.Close()
		c := ping.NewPingClient(conn)
		p.clients[port] = c
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		p.sendPingToAll()

		p.mu.Lock()

		print("Doing important work.")
		time.Sleep(1 * time.Second)
		print("...")
		time.Sleep(2 * time.Second)
		print("Success!")

		p.mu.Unlock()
	}
}

type peer struct {
	ping.UnimplementedPingServer
	id              int32
	sequence_number int32
	clients         map[int32]ping.PingClient
	ctx             context.Context
	mu              sync.Mutex
}

func (p *peer) Ping(ctx context.Context, req *ping.Request) (*ping.Reply, error) {
	id := req.Id

	rep := &ping.Reply{Flag: 1}
	return rep, nil
}

func (p *peer) sendPingToAll() int32 {
	request := &ping.Request{Id: p.id}
	for id, client := range p.clients {
		reply, err := client.Ping(p.ctx, request)
		if err != nil {
			fmt.Println("something went wrong")
			// Error handling
			return 0
		}
		fmt.Printf("Got reply from id %v: %v\n", id, reply.Flag)
	}
	return 1
}
