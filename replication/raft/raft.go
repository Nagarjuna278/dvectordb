package raft

import (
	"context"
	pb "dvecdb/proto"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Log struct {
	index int
	value []byte
}

type PeerClient struct {
	nodelist   *pb.NodeList
	conn       *grpc.ClientConn
	raftClient pb.NodeRegistrationClient
	Address    string
}
type State int

const (
	Leader State = iota
	Candidate
	Follower
)

type RaftServer struct {
	pb.UnimplementedNodeRegistrationServer
	mu               sync.Mutex
	Id               int
	state            State
	Addr             string
	Peers            map[string]*PeerClient
	Term             int
	Index            int
	logs             []Log
	lastLog          Log
	Ctx              context.Context
	electionTimeout  time.Duration
	heatbeatInterval time.Duration
	electionTicker   *time.Ticker
	heartbeatTicker  *time.Ticker
	votesRecieved    int
	grpcServer       *grpc.Server
	leaderAddr       string
}

type Server struct {
	*pb.UnimplementedNodeRegistrationServer
	mu       sync.Mutex
	raftServ *rpc.Server
}

func NewNode(id int, addr string) *RaftServer {
	node := &RaftServer{
		Id:               id,
		Addr:             addr,
		Peers:            map[string]*PeerClient{},
		Term:             0,
		Index:            0,
		logs:             []Log{},
		lastLog:          Log{},
		state:            Follower,
		electionTimeout:  300 * time.Millisecond,
		heatbeatInterval: 150 * time.Millisecond,
		votesRecieved:    0,
	}
	return node
}

func (rs *RaftServer) Run(peers []string) {
	if rs.Addr == "localhost:5001" {
		rs.state = Leader
	}
	grpcServer := grpc.NewServer()
	pb.RegisterNodeRegistrationServer(grpcServer, rs)
	lis, err := net.Listen("tcp", rs.Addr)
	if err != nil {
		fmt.Println("Tcp Connection error")
	}
	go grpcServer.Serve(lis)
	rs.GenerateRPCconnection(peers)
	go rs.InitRaftServer()
	for _, val := range rs.Peers {
		fmt.Println(rs.Id, val)
	}
}

func (rs *RaftServer) InitRaftServer() {
	go rs.electionTimerLoop()
	for {
		rs.mu.Lock()
		state := rs.state
		rs.mu.Unlock()
		switch state {
		case Leader:
			rs.RunLeader()
		case Follower:
			rs.RunFollower()
		case Candidate:
			rs.RunCandidate()
		}
	}
}
func (rs *RaftServer) resetElectionTimeout() {
	newTimeout := 300*time.Millisecond + time.Duration(rand.Intn(150))*time.Millisecond
	rs.electionTicker.Reset(newTimeout)
}
func (rs *RaftServer) StartElection() {
	fmt.Println("election Triggered")
	rs.state = Candidate
	votes := 0
	for id, peer := range rs.Peers {
		if id == strconv.Itoa(rs.Id) {
			votes++
			continue
		}
		go func(peer *PeerClient) {
			resp, _ := peer.raftClient.RequestVote(context.Background(), &pb.RequestVoteArgs{CandidateId: id, CandidateTerm: int32(rs.Term) + 1, Addr: rs.Addr})
			if resp.VoteGranted == true {
				votes++
			}
		}(peer)
	}
	if votes > 1 {
		rs.state = Leader
		rs.votesRecieved = votes
		rs.leaderAddr = rs.Addr
	}
}
func (rs *RaftServer) electionTimerLoop() {
	fmt.Println("inside electionTimerLoop")
	rs.electionTicker = time.NewTicker(rs.electionTimeout)
	rs.heartbeatTicker = time.NewTicker(rs.heatbeatInterval + time.Duration(rand.Intn(50)*int(time.Millisecond)))
	for {
		select {
		case <-rs.electionTicker.C:
			rs.mu.Lock()
			fmt.Println("starting election")
			if rs.state != Leader {
				rs.StartElection()
			}
			fmt.Println("ending election")
			rs.mu.Unlock()
		case <-rs.heartbeatTicker.C:
			rs.mu.Lock()
			if rs.state == Leader {
				rs.RunLeader()
			}
			rs.mu.Unlock()
			rs.resetElectionTimeout()
		}
	}
}

func (rs *RaftServer) RegisterNode(ctx context.Context, nodeinfo *pb.NodeInfo) (*pb.RegisterResponse, error) {
	return &pb.RegisterResponse{Message: "Success"}, nil
}

func (rs *RaftServer) GetNodes(ctx context.Context, empty *pb.Empty) (*pb.NodeList, error) {
	return &pb.NodeList{}, nil
}

func (rs *RaftServer) SendHeartBeat(ctx context.Context, signal *pb.Signal) (*pb.Signal, error) {
	return &pb.Signal{Flag: true}, nil
}

func (rs *RaftServer) RequestVote(ctx context.Context, request *pb.RequestVoteArgs) (*pb.RequestVoteReply, error) {
	if request.CandidateTerm > int32(rs.Term) {
		fmt.Println("this is ", rs.Id)
		rs.Term = int(request.CandidateTerm)
		rs.state = Follower
		rs.leaderAddr = request.Addr
		return &pb.RequestVoteReply{VoteGranted: true}, nil
	}
	return &pb.RequestVoteReply{VoteGranted: false}, nil
}

func (rs *RaftServer) GenerateRPCconnection(peers []string) {
	for _, peerAddr := range peers {
		if peerAddr == rs.Addr {
			continue
		} else {
			if _, exists := rs.Peers[strconv.Itoa(rs.Id)]; exists {
				return
			}
			conn, err := grpc.NewClient(peerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			client := pb.NewNodeRegistrationClient(conn)
			resp, err := client.RegisterNode(context.Background(), &pb.NodeInfo{Id: strconv.Itoa(rs.Id), Addr: rs.Addr})
			if err != nil {
				fmt.Println(err)
			}
			rs.Peers[strconv.Itoa(rs.Id)] = &PeerClient{
				nodelist:   &pb.NodeList{},
				conn:       conn,
				raftClient: client,
				Address:    rs.Addr,
			}
			fmt.Println(resp.Message)
		}
	}

}

func (rs *RaftServer) RunLeader() {
	for id, peers := range rs.Peers {
		if id == strconv.Itoa(rs.Id) {
			continue
		}

		resp, err := peers.raftClient.SendHeartBeat(context.Background(), &pb.Signal{Flag: true})
		if err != nil {
			fmt.Println(id, "error", err)
		}
		fmt.Println(resp.Flag, id)
	}
}
