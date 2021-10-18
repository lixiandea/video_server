package main

import "log"

type ConnLimiter struct {
	numConn int
	bucket  chan int
}

func NewLimiter(nc int) *ConnLimiter {
	return &ConnLimiter{
		numConn: nc,
		bucket:  make(chan int, nc),
	}
}

func (l *ConnLimiter) getToken() bool {
	if len(l.bucket) >= l.numConn {
		log.Printf("Reached the rate limitation.")
		return false
	}
	l.bucket <- 1
	return true
}

func (l *ConnLimiter) releaseToken() {
	c := <-l.bucket
	log.Printf("Release conn %d.", c)
}
