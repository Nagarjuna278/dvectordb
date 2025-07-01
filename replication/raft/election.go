package raft

import (
	"context"
	pb "dvecdb/proto"
	"fmt"
	"sync"
	"time"
)

func (rs *RaftServer) RunFollower() {
	elecTimeout := time.NewTimer(generateRandomTimeout())
	defer elecTimeout.Stop()
	for {
		select {
		case <-rs.heartbeatCh:
			if !elecTimeout.Stop() {
				<-elecTimeout.C
			}
			elecTimeout.Reset(generateRandomTimeout())
		case <-elecTimeout.C:
			rs.mu.Lock()
			rs.state = Candidate
			rs.mu.Unlock()
			return
		case <-rs.ctx.Done():
			rs.DeRegister()
			return
		}
	}
}
func (rs *RaftServer) RunCandidate() {
	// fmt.Println("inside the runcandidate", rs.Addr, rs.Term)
	timer := time.NewTimer(generateRandomTimeout())
	defer func() {
		// fmt.Println("exiting the candidate", rs.Id, rs.Term, rs.state)
	}()
	for {
		// 	 Id            string                 `protobuf:"bytes,1,opt,name=Id,proto3" json:"Id,omitempty"`
		// Term          int32                  `protobuf:"varint,2,opt,name=Term,proto3" json:"Term,omitempty"`
		// LogInd        int32                  `protobuf:"varint,3,opt,name=LogInd,proto3" json:"LogInd,omitempty"`
		// Addr          string
		rs.mu.Lock()
		rs.Term++
		// fmt.Println("eleciton starting by candidate", rs.Id)
		n := len(rs.Peers)
		peers := rs.Peers
		rs.VotedFor = rs.Addr
		rs.mu.Unlock()
		voteresponse := make(chan bool, n)
		votes := 1
		var wg sync.WaitGroup
		leaderch := make(chan struct{}, 1)
		for _, peer := range peers {
			wg.Add(1)
			go func(peer *PeerClient) {
				defer wg.Done()
				rs.mu.Lock()
				req := &pb.RequestVoteArgs{Id: rs.Id, Term: int32(rs.Term), Addr: rs.Addr}
				rs.mu.Unlock()
				resp, err := peer.raftNode.RequestVote(context.Background(), req)
				if err != nil {
					fmt.Println(err)
					return
				}
				// 			message RequestVoteReply {
				// bool vote_granted = 1;
				// int32 term=2;
				// string votedFor=3;
				// }
				// fmt.Println(resp.VoteGranted, resp.Term, resp.VotedFor, req.Id, peer.Addr)

				if resp.Term > req.Term {
					rs.heartbeatCh <- struct{}{}
					return
				} else if resp.Term == req.Term && resp.VoteGranted {
					voteresponse <- true
				} else {
					voteresponse <- false
				}
			}(peer)
		}
		go func() {
			wg.Wait()
			close(voteresponse)
		}()
		for {
			select {
			case <-rs.heartbeatCh:
				rs.mu.Lock()
				rs.state = Follower
				rs.mu.Unlock()
				return
			case voteres, ok := <-voteresponse:
				if !ok {
					voteresponse = nil
					return
				}
				if ok && voteres {
					votes++
					// fmt.Println(votes)
					if votes > n/2 {
						leaderch <- struct{}{}
					}
				}
			case <-leaderch:
				rs.mu.Lock()
				rs.state = Leader
				rs.mu.Unlock()
				return
			case <-timer.C:
				break
			}
		}

	}
}

func (rs *RaftServer) RunLeader() {
	newTimer := time.NewTimer(4 * time.Second)
	shouldStepDown := make(chan bool, 1)

	heartbeat := func() {
		var wg sync.WaitGroup

		for _, peer := range rs.Peers {
			wg.Add(1)
			go func(p *PeerClient) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()
				rs.mu.Lock()
				defer rs.mu.Unlock()

				req := &pb.HearBeatRequest{
					Id:     rs.Id,
					Term:   int32(rs.Term),
					LogInd: int32(rs.logInd),
					Add:    rs.Addr,
				}
				resp, err := p.raftNode.SendHeartBeat(ctx, req)
				if err != nil {
					return
				}
				if resp.Flag == true {
					return
				}
				if resp.Term > req.Term || resp.NewLeader == true {
					rs.state = Follower
					rs.Term = int(resp.Term)
					shouldStepDown <- true
				}
			}(peer)
		}

		wg.Wait()
	}
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	rs.mu.Lock()
	isLeader := (rs.state == Leader)
	if isLeader {
		fmt.Println("current leader", rs.Addr)

	}
	rs.mu.Unlock()

	if !isLeader {
		return // Step down if state changed
	}
	heartbeat()

	for {
		select {
		case <-ticker.C:
			rs.mu.Lock()
			isLeader := (rs.state == Leader)
			// fmt.Println(rs.Addr, isLeader)
			rs.mu.Unlock()

			if !isLeader {
				return // Step down if state changed
			}
			heartbeat()
		case <-shouldStepDown:
			return
		case <-newTimer.C:
			rs.mu.Lock()
			rs.state = Follower
			fmt.Println("breaking Leader")
			rs.mu.Unlock()
		case <-rs.ctx.Done():
			rs.DeRegister()
			return
		}
	}
}
