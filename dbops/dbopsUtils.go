package dbops

import (
	"log"
	"strconv"
	"time"
)

func GetTimeStamp() int {
	ts, err := strconv.Atoi(strconv.FormatInt(time.Now().UnixNano()/1000000000, 10))
	if err != nil {
		log.Printf("Get TimeStamp failed: %v", err)
		return 0
	}
	return ts
}
