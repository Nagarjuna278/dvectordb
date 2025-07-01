package raft

import (
	"math/rand"
	"time"
)

func generateRandomTimeout() time.Duration {
	return (time.Duration(electionTimeoutMin) + time.Duration(rand.Intn(400))*time.Millisecond)
}
