package raft

import (
	"context"
	pb "dvecdb/proto"
)

// message RequestVoteArgs {
//   string Id = 1;
//   int32 Term = 2;
//   int32 LogInd = 3;
//   string Addr = 4;
// }

//	message RequestVoteReply {
//	  bool vote_granted = 1;
//	  int32 term=2;
//	  string votedFor=3;
//	  int32 logInd = 4;
//	}
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
			resp.VotedFor = req.Id
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
		rs.leaderId = req.Id
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
