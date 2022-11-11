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
		p.requesting_critical = true

		flag := p.sendPingToAll()
		if flag == 1 {
			p.requesting_critical = false
			continue
		}

		p.mu.Lock()

		print("Doing important work.")
		time.Sleep(1 * time.Second)
		print("...")
		time.Sleep(2 * time.Second)
		print("Success!")

		p.mu.Unlock()

		p.requesting_critical = false
	}
}

type peer struct {
	ping.UnimplementedPingServer
	id                  int32
	sequence_number     int64
	clients             map[int32]ping.PingClient
	ctx                 context.Context
	requesting_critical bool
	mu                  sync.Mutex
}

func (p *peer) Ping(ctx context.Context, req *ping.Request) (*ping.Reply, error) {
	id := req.Id

	if p.requesting_critical {
		if req.sequence_number > p.sequence_number {
			for p.requesting_critical {
				time.Sleep(0.1 * time.Seconds())
			}
		} else if req.sequence_number == p.sequence_number && req.id > p.id {
			for p.requesting_critical {
				time.Sleep(0.1 * time.Seconds())
			}
		}
	}

	rep := &ping.Reply{flag: 1}
	return rep, nil
}

func (p *peer) sendPingToAll() int32 {
	// Little bit of a cheat
	p.sequence_number = time.Now().UnixNano()

	request := &ping.Request{Id: p.id, sequence_number: p.sequence_number}
	for id, client := range p.clients {
		reply, err := client.Ping(p.ctx, request)
		if err != nil {
			fmt.Println("Another client has died. Abort mission.")
			// Error handling
			return 0
		}
		fmt.Printf("Got reply from id %v: %v\n", id, reply.flag)
	}
	return 1
}
