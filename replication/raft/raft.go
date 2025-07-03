package raft

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "dvecdb/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewNode(id int, addr string) *RaftServer {
	node := &RaftServer{
		Id:            id,
		Addr:          addr,
		Term:          0,
		logInd:        0,
		log:           make([]*Log, 0),
		lasLogTerm:    0,
		VotedFor:      "",
		state:         Follower,
		elecTimeoutCh: make(chan struct{}),
		heartbeatCh:   make(chan struct{}),
		Peers:         make(map[string]*PeerClient),
		nextIndex:     make(map[string]int, 0),
		matchIndex:    make(map[string]int, 0),
	}
	ctx, cancel := context.WithCancel(context.Background())
	node.ctx = ctx
	node.cancel = cancel
	node.createGrpcServer()
	return node
}

func (rs *RaftServer) createGrpcServer() {
	lis, err := net.Listen("tcp", rs.Addr)
	if err != nil {
		log.Println(err)
	}
	grpcSrv := grpc.NewServer()
	rs.grpcSrv = grpcSrv
	pb.RegisterNodeRegistrationServer(rs.grpcSrv, rs)
	fmt.Println("done creating the grpc server for the node", rs.Id)

	go grpcSrv.Serve(lis)
}
func (rs *RaftServer) CreateClients(peers []string) {
	for _, x := range peers {
		if x == rs.Addr {
			continue
		}
		rs.Peers[x] = &PeerClient{
			Addr: x,
		}
		conn, err := grpc.NewClient(x, grpc.WithTransportCredentials(insecure.NewCredentials()))
		res := pb.NewNodeRegistrationClient(conn)

		if err != nil {
			continue
		}
		rs.Peers[x].raftNode = res
	}
}

func (rs *RaftServer) Run(peers []string) {
	go rs.CreateClients(peers)
	go rs.InitRaft()
}

func (rs *RaftServer) DeRegister() {

}

func (rs *RaftServer) InitRaft() {
	for {
		select {
		case <-rs.ctx.Done():
			rs.DeRegister()
			return
		default:
			rs.mu.Lock()
			state := rs.state
			id := rs.Id
			rs.mu.Unlock()
			switch state {
			case Follower:
				fmt.Println("Follower", id)
				rs.RunFollower()
			case Candidate:
				fmt.Println("Candidate", id)
				rs.RunCandidate()
			case Leader:
				fmt.Println("Leader", id)
				rs.RunLeader()
			}
		}
	}
}
