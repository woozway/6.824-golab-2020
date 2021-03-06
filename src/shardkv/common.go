package shardkv

import "../labgob"

//
// Sharded key/value server.
// Lots of replica groups, each running op-at-a-time paxos.
// Shardmaster decides which group serves each shard.
// Shardmaster may change shard assignment from time to time.
//
// You will have to modify these definitions.
//

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongGroup  = "ErrWrongGroup"
	ErrWrongLeader = "ErrWrongLeader"
	ErrTimeOut     = "ErrTimeOut"
)

type Err string

func init() {
	labgob.Register(PutAppendArgs{})
	labgob.Register(PutAppendReply{})
	labgob.Register(GetArgs{})
	labgob.Register(GetReply{})
	labgob.Register(FetchShardDataArgs{})
	labgob.Register(FetchShardDataReply{})
	labgob.Register(CleanShardDataArgs{})
	labgob.Register(CleanShardDataReply{})
	labgob.Register(MergeShardData{})
}

// Put or Append
type PutAppendArgs struct {
	// You'll have to add definitions here.
	Key   string
	Value string
	Op    string // "Put" or "Append"
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	ClientId  int64
	MsgId     int64
	ConfigNum int
}

type PutAppendReply struct {
	Err Err
}

func (c *PutAppendArgs) copy() PutAppendArgs {
	res := PutAppendArgs{}
	res.Key = c.Key
	res.Value = c.Value
	res.Op = c.Op
	res.ClientId = c.ClientId
	res.MsgId = c.MsgId
	res.ConfigNum = c.ConfigNum
	return res
}

type GetArgs struct {
	Key       string
	ClientId  int64
	MsgId     int64
	ConfigNum int
}

type GetReply struct {
	Err   Err
	Value string
}

func (c *GetArgs) copy() GetArgs {
	res := GetArgs{}
	res.Key = c.Key
	res.ClientId = c.ClientId
	res.MsgId = c.MsgId
	res.ConfigNum = c.ConfigNum
	return res
}

type FetchShardDataArgs struct {
	ConfigNum int
	ShardNum  int
}

type FetchShardDataReply struct {
	Success    bool
	MsgIndexes map[int64]int64
	Data       map[string]string
}

func (reply *FetchShardDataReply) copy() FetchShardDataReply {
	res := FetchShardDataReply{}
	res.Success = reply.Success
	res.Data = make(map[string]string)
	res.MsgIndexes = make(map[int64]int64)
	for k, v := range reply.Data {
		res.Data[k] = v
	}
	for k, v := range reply.MsgIndexes {
		res.MsgIndexes[k] = v
	}
	return res
}

type CleanShardDataArgs struct {
	ConfigNum int
	ShardNum  int
}

type CleanShardDataReply struct {
	Success bool
}

type MergeShardData struct {
	ConfigNum  int
	ShardNum   int
	MsgIndexes map[int64]int64
	Data       map[string]string
}
