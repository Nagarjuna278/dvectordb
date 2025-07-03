package raft

import (
	"context"
	pb "dvecdb/proto"
)

// Arguments:
// term candidate’s term
// candidateId candidate requesting vote
// lastLogIndex index of candidate’s last log entry (§5.4)
// lastLogTerm term of candidate’s last log entry (§5.4)

// Results:
// term currentTerm, for candidate to update itself
// voteGranted true means candidate received vote
// Receiver implementation:
// 1. Reply false if term < currentTerm (§5.1)
// 2. If votedFor is null or candidateId, and candidate’s log is at
// least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)
func (rs *RaftServer) RequestVote(ctx context.Context, req *pb.RequestVoteArgs) (*pb.RequestVoteReply, error) {
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		rs.mu.Lock()
		resp := &pb.RequestVoteReply{VoteGranted: false, Term: int32(rs.Term), VotedFor: rs.VotedFor}
		defer rs.mu.Unlock()
		if resp.Term > req.Term {
			return resp, nil
		} else if resp.Term == req.Term && (resp.VotedFor != "" || resp.VotedFor == req.Addr) {
			resp.VoteGranted = true
			resp.VotedFor = req.Addr
		} else if resp.Term < req.Term {
			resp.VoteGranted = true
			resp.Term = req.Term
			resp.VotedFor = req.Addr
		}
		return resp, nil
	}
}

func (rs *RaftServer) SendHeartBeat(ctx context.Context, req *pb.HearBeatRequest) (*pb.HearBeatReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.state == Candidate && rs.Term == int(req.Term) {
		rs.state = Follower
		rs.leaderId = int(req.Id)
		return &pb.HearBeatReply{Flag: true, NewLeader: false, Term: int32(rs.Term)}, nil
	}

	if rs.Term > int(req.Term) {
		return &pb.HearBeatReply{Flag: false, NewLeader: true, Term: int32(rs.Term)}, nil
	}
	rs.Term = int(req.Term)
	rs.heartbeatCh <- struct{}{}
	return &pb.HearBeatReply{Flag: true, NewLeader: false, Term: req.Term}, nil
}

func (rs *RaftServer) GetNodes(ctx context.Context, emp *pb.Empty) (*pb.NodeList, error) {
	return &pb.NodeList{}, nil
}
func (rs *RaftServer) RegisterNode(ctx context.Context, node *pb.NodeInfo) (*pb.RegisterResponse, error) {
	return &pb.RegisterResponse{Message: "Successful"}, nil
}

func (rs *RaftServer) CheckRPC(ctx context.Context, empty *pb.Empty) (*pb.RegisterResponse, error) {
	return &pb.RegisterResponse{Message: "working"}, nil
}

func (rs *RaftServer) AppendEntries(ctx context.Context, req *pb.AppendEntriesArgs) (*pb.AppendEntriesReply, error) {
	// type AppendEntriesArgs struct {
	// state         protoimpl.MessageState `protogen:"open.v1"`
	// Term          string                 `protobuf:"bytes,1,opt,name=Term,proto3" json:"Term,omitempty"`
	// Id            string                 `protobuf:"bytes,2,opt,name=Id,proto3" json:"Id,omitempty"`
	// PrevLogIndex  int32                  `protobuf:"varint,3,opt,name=prevLogIndex,proto3" json:"prevLogIndex,omitempty"`
	// PrevLogTerm   int32                  `protobuf:"varint,4,opt,name=prevLogTerm,proto3" json:"prevLogTerm,omitempty"`
	// Log           []*Log                 `protobuf:"bytes,5,rep,name=log,proto3" json:"log,omitempty"`
	rs.mu.Lock()
	defer rs.mu.Unlock()
	resp := &pb.AppendEntriesReply{Term: int32(rs.Term), Success: false, PrevLogInd: 0, PrevLogTerm: 0}
	if req.PrevLogIndex == 0 {
		rs.log = rs.log[:0]
		for _, v := range req.Log {
			rs.log = append(rs.log, &Log{message: v.Command, Term: int(v.Term), Index: int(v.Ind)})
		}
		resp.Term = req.Term
		resp.Success = true
	} else if req.PrevLogIndex > int32(len(rs.log)) {
		if len(rs.log) > 0 {
			resp.PrevLogInd = int32(len(rs.log) - 1)
			resp.PrevLogTerm = int32(rs.log[len(rs.log)-1].Term)
		}
	} else if req.PrevLogTerm != int32(rs.log[req.PrevLogIndex-1].Term) {
		resp.PrevLogInd = req.PrevLogIndex - 1
		if len(rs.log) > 0 {
			resp.PrevLogTerm = int32(rs.log[len(rs.log)-1].Term)
		}
	} else {
		rs.log = rs.log[:req.PrevLogIndex]
		for _, v := range req.Log {
			rs.log = append(rs.log, &Log{message: v.Command, Term: int(v.Term), Index: int(v.Ind)})
		}
		resp.Success = true
	}
	rs.heartbeatCh <- struct{}{}
	return resp, nil
}
