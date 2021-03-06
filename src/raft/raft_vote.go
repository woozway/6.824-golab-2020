package raft

import "time"

type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	lastLogTerm, lastLogIndex := rf.lastLogTermIndex()
	reply.Term = rf.term
	reply.VoteGranted = false

	if args.Term < rf.term {
		return
	} else if args.Term == rf.term {
		if rf.role == Leader {
			return
		}
		if rf.votedFor == args.CandidateId {
			reply.VoteGranted = true
			return
		}
		if rf.votedFor != -1 && rf.votedFor != args.CandidateId {
			// already voted
			return
		}
	}

	defer rf.persist()
	if args.Term > rf.term {
		rf.term = args.Term
		rf.votedFor = -1
		rf.changeRole(Follower)
	}

	if lastLogTerm > args.LastLogTerm || (args.LastLogTerm == lastLogTerm && args.LastLogIndex < lastLogIndex) {
		return
	}

	rf.term = args.Term
	rf.votedFor = args.CandidateId
	reply.VoteGranted = true
	rf.changeRole(Follower)
	rf.resetElectionTimer()
}

func (rf *Raft) resetElectionTimer() {
	rf.electionTimer.Stop()
	rf.electionTimer.Reset(randElectionTimeout())
}

func (rf *Raft) sendRequestVoteToPeer(server int, args *RequestVoteArgs, reply *RequestVoteReply) {
	t := time.NewTimer(RPCTimeout)
	defer t.Stop()
	rpcTimer := time.NewTimer(RPCTimeout)
	defer rpcTimer.Stop()

	for {
		rpcTimer.Stop()
		rpcTimer.Reset(RPCTimeout)
		ch := make(chan bool, 1)
		r := RequestVoteReply{}

		go func() {
			ok := rf.peers[server].Call("Raft.RequestVote", args, &r)
			if !ok {
				time.Sleep(time.Millisecond * 10)
			}
			ch <- ok
		}()

		select {
		case <-t.C:
			return
		case <-rpcTimer.C:
			continue
		case ok := <-ch:
			if !ok {
				continue
			} else {
				reply.Term = r.Term
				reply.VoteGranted = r.VoteGranted
				return
			}
		}
	}
}

func (rf *Raft) startElection() {
	rf.mu.Lock()
	rf.electionTimer.Reset(randElectionTimeout())
	if rf.role == Leader {
		rf.mu.Unlock()
		return
	}
	rf.changeRole(Candidate)
	lastLogTerm, lastLogIndex := rf.lastLogTermIndex()
	args := RequestVoteArgs{}
	args.Term = rf.term
	args.CandidateId = rf.me
	args.LastLogIndex = lastLogIndex
	args.LastLogTerm = lastLogTerm
	rf.persist()
	rf.mu.Unlock()

	grantedCount := 1
	votesCount := 1
	votesCh := make(chan bool, len(rf.peers))
	for index := range rf.peers {
		if index != rf.me {
			go func(ch chan bool, index int) {
				reply := RequestVoteReply{}
				rf.sendRequestVoteToPeer(index, &args, &reply)
				ch <- reply.VoteGranted
				if reply.Term > args.Term {
					rf.mu.Lock()
					if rf.term < reply.Term {
						rf.term = reply.Term
						rf.changeRole(Follower)
						rf.votedFor = -1
						rf.resetElectionTimer()
						rf.persist()
					}
					rf.mu.Unlock()
				}
			}(votesCh, index)
		}
	}

	for {
		voteInFavour := <-votesCh
		votesCount += 1
		if voteInFavour {
			grantedCount += 1
		}
		if votesCount == len(rf.peers) || grantedCount > len(rf.peers)/2 || votesCount-grantedCount > len(rf.peers)/2 {
			break
		}
	}

	if grantedCount <= len(rf.peers)/2 {
		return
	}

	rf.mu.Lock()
	if rf.term == args.Term && rf.role == Candidate {
		rf.changeRole(Leader)
		rf.persist()
	}
	if rf.role == Leader {
		rf.resetHeartbeatTimers()
	}
	rf.mu.Unlock()
}
