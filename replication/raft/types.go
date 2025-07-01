package raft

import (
	"context"
	pb "dvecdb/proto"
	"sync"
	"time"

	"google.golang.org/grpc"
)

type Log struct {
	message string
	Index   int
	Term    int
}
type State int

const (
	Follower State = iota
	Candidate
	Leader
)

const (
	electionTimeoutInterval = 1 * time.Second
	electionTimeoutMin      = 1200 * time.Millisecond
	electionTimeoutMax      = 800 * time.Millisecond
	heartbeatInterval       = 500 * time.Millisecond
)

type PeerClient struct {
	Addr     string
	done     chan struct{}
	raftNode pb.NodeRegistrationClient
}

type RaftServer struct {
	pb.UnimplementedNodeRegistrationServer
	mu     sync.Mutex
	Id     string
	Addr   string
	Term   int
	logInd int
	log    []Log

	lasLogTerm    int
	VotedFor      string
	leaderId      string
	state         State
	elecTimeoutCh chan struct{}
	heartbeatCh   chan struct{}
	ctx           context.Context
	grpcSrv       *grpc.Server
	cancel        context.CancelFunc
	Peers         map[string]*PeerClient
}
