package dbops

import (
	"strconv"
	"time"
)

func GetTimeStamp() (int, error) {
	ts, err := strconv.Atoi(strconv.FormatInt(time.Now().UnixNano()/1000000000, 10))
	return ts, err
}
