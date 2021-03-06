package raft

import (
	"log"
	"time"
)

type InstallSnapshotArgs struct {
	Term              int
	LeaderId          int
	LastIncludedIndex int
	LastIncludedTerm  int
	Data              []byte
}

type InstallSnapshotReply struct {
	Term int
}

func (rf *Raft) SavePersistSnapshot(logIndex int, snapshot []byte) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if logIndex <= rf.lastSnapshotIndex {
		return
	}
	if logIndex > rf.commitIndex {
		panic("logindex > rf.commitdex")
	}

	// logindex must be in raft.logEntries
	lastLog := rf.getLogByIndex(logIndex)
	rf.logEntries = rf.logEntries[rf.getRealIdxByLogIndex(logIndex):]
	rf.lastSnapshotIndex = logIndex
	rf.lastSnapshotTerm = lastLog.Term
	state := rf.getPersistState()
	rf.persister.SaveStateAndSnapshot(state, snapshot)
}

func (rf *Raft) sendInstallSnapshot(peerIdx int) {
	rf.mu.Lock()
	args := InstallSnapshotArgs{}
	args.Term = rf.term
	args.LeaderId = rf.me
	args.LastIncludedIndex = rf.lastSnapshotIndex
	args.LastIncludedTerm = rf.lastSnapshotTerm
	args.Data = rf.persister.ReadSnapshot()
	rf.mu.Unlock()

	timer := time.NewTimer(RPCTimeout)
	defer timer.Stop()
	for {
		timer.Stop()
		timer.Reset(RPCTimeout)
		okCh := make(chan bool, 1)
		reply := InstallSnapshotReply{}
		go func() {
			o := rf.peers[peerIdx].Call("Raft.InstallSnapshot", &args, &reply)
			if !o {
				time.Sleep(time.Millisecond * 10)
			}
			okCh <- o
		}()

		ok := false
		select {
		case <-rf.stopCh:
			return
		case <-timer.C:
			continue
		case ok = <-okCh:
			if !ok {
				continue
			}
		}

		// ok == true
		rf.mu.Lock()
		defer rf.mu.Unlock()
		if rf.term != args.Term || rf.role != Leader {
			return
		}
		if reply.Term > rf.term {
			rf.changeRole(Follower)
			rf.resetElectionTimer()
			rf.term = reply.Term
			rf.persist()
			return
		}
		// success
		if args.LastIncludedIndex > rf.matchIndex[peerIdx] {
			rf.matchIndex[peerIdx] = args.LastIncludedIndex
		}
		if args.LastIncludedIndex+1 > rf.nextIndex[peerIdx] {
			rf.nextIndex[peerIdx] = args.LastIncludedIndex + 1
		}
		return
	}
}

func (rf *Raft) InstallSnapshot(args *InstallSnapshotArgs, reply *InstallSnapshotReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.term
	if args.Term < rf.term {
		return
	}
	if args.Term > rf.term || rf.role != Follower {
		rf.term = args.Term
		rf.changeRole(Follower)
		rf.resetElectionTimer()
		defer rf.persist()
	}
	if rf.lastSnapshotIndex >= args.LastIncludedIndex {
		return
	}
	// success
	start := args.LastIncludedIndex - rf.lastSnapshotIndex
	if start < 0 {
		log.Fatal("install sn")
	} else if start >= len(rf.logEntries) {
		rf.logEntries = make([]LogEntry, 1)
		rf.logEntries[0].Term = args.LastIncludedTerm
	} else {
		rf.logEntries = rf.logEntries[start:]
	}

	rf.lastSnapshotIndex = args.LastIncludedIndex
	rf.lastSnapshotTerm = args.LastIncludedTerm
	rf.persister.SaveStateAndSnapshot(rf.getPersistState(), args.Data)
}
