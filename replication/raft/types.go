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
	Id     int
	Addr   string
	Term   int
	logInd int
	log    []*Log

	lasLogTerm int
	VotedFor   string
	leaderId   int
	state      State

	commitIndex int // Index of highest log entry known to be committed
	lastApplied int

	nextIndex  map[string]int // For each follower, index of the next log entry to send
	matchIndex map[string]int // For each follower, index of the highest log entry replicated

	elecTimeoutCh chan struct{}
	heartbeatCh   chan struct{}
	applyCh       chan ApplyCmd
	ctx           context.Context
	grpcSrv       *grpc.Server
	cancel        context.CancelFunc
	Peers         map[string]*PeerClient
}

type ApplyCmd struct {
	CmdValid bool
	Cmd      string
	Key      string
	Value    string
	CmdInd   int
	CmdTerm  int
}

func (l *Log) convertToProto() *pb.Log {
	return &pb.Log{
		Ind:     int32(l.Index),
		Term:    int32(l.Term),
		Command: l.message,
		Key:     "",
		Value:   "",
	}
}

type Loglist struct {
	log []*Log
}

func (l *Loglist) convertLogSliceToProto() []*pb.Log {
	loglist := make([]*pb.Log, 0)
	for _, value := range l.log {
		loglist = append(loglist, &pb.Log{Ind: int32(value.Index),
			Term:    int32(value.Term),
			Command: value.message,
			Key:     "",
			Value:   "",
		})
	}
	return loglist
}
