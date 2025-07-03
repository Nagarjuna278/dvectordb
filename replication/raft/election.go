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
				lastLogInd := 0
				lastLogTerm := 0
				if len(rs.log) > 0 {
					lastLogInd = rs.log[len(rs.log)-1].Index
					lastLogTerm = rs.log[len(rs.log)-1].Term
				}
				req := &pb.RequestVoteArgs{Id: int32(rs.Id), Term: int32(rs.Term), Addr: rs.Addr, LastLogInd: int32(lastLogInd), LastLogTerm: int32(lastLogTerm)}
				rs.mu.Unlock()
				resp, err := peer.raftNode.RequestVote(context.Background(), req)
				if err != nil {
					fmt.Println(err)
					return
				}
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
				lastlogInd := len(rs.log) - 1
				for i, _ := range rs.Peers {
					rs.nextIndex[i] = lastlogInd
					rs.matchIndex[i] = 0
				}
				rs.mu.Unlock()
				return
			case <-timer.C:
				break
			}
		}

	}
}

func (rs *RaftServer) RunLeader() {
	ticker := time.NewTicker(heartbeatInterval)
	commitChan := make(chan int, 1)
	n := len(rs.Peers)
	voteresp := make(chan bool, n)
	applyMsg := make(chan ApplyCmd, 1)
	votes := 1
	heartbeat := func() {
		for addr, peer := range rs.Peers {
			go func(addr string, peer *PeerClient) {
				req := &pb.AppendEntriesArgs{
					// 				Term          int32                  `protobuf:"varint,1,opt,name=Term,proto3" json:"Term,omitempty"`
					// Id            string                 `protobuf:"bytes,2,opt,name=Id,proto3" json:"Id,omitempty"`
					// PrevLogIndex  int32                  `protobuf:"varint,3,opt,name=prevLogIndex,proto3" json:"prevLogIndex,omitempty"`
					// PrevLogTerm   int32                  `protobuf:"varint,4,opt,name=prevLogTerm,proto3" json:"prevLogTerm,omitempty"`
					// Log           []*Log                 `protobuf:"bytes,5,rep,name=log,proto3
					Term:         int32(rs.Term),
					Id:           int32(rs.Id),
					PrevLogIndex: 0,
					PrevLogTerm:  0,
				}
				retry := 0
				for retry < 3 {
					if rs.nextIndex[addr] >= 0 {
						req.PrevLogIndex = int32(rs.nextIndex[addr] - 1)
						req.PrevLogTerm = int32(rs.log[rs.nextIndex[addr]].Term)
					}
					reqLog := rs.log[rs.nextIndex[addr]:]
					pblog := (&Loglist{log: reqLog}).convertLogSliceToProto()
					req.Log = pblog
					resp, err := rs.AppendEntries(context.Background(), req)
					if err != nil {
						fmt.Println(err)
					}
					if resp.Success {
						voteresp <- true
						break
					} else {
						rs.nextIndex[addr] = int(resp.PrevLogInd)
						retry++
					}
				}

			}(addr, peer)
		}
	}
	for {
		select {
		case msg := <-applyMsg:
			rs.log = append(rs.log, &Log{message: msg.Cmd, Term: rs.Term, Index: len(rs.log)})
			heartbeat()
		case vote := <-voteresp:
			if vote {
				votes++
				if votes > n/2 {
					commitChan <- rs.log[len(rs.log)-1].Index
				}
			}
		case logInd := <-commitChan:
			rs.commitIndex = logInd
		case <-ticker.C:
			heartbeat()
			// pass
		case <-rs.ctx.Done():
			rs.DeRegister()
			return
		}
	}
}
