package session

import (
	"github.com/lixiandea/video_server/dbops"
	"github.com/lixiandea/video_server/entity"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
)

var sessionMap *sync.Map

func init() {
	sessionMap = &sync.Map{}
}

func nowInMilli() int64 {
	return time.Now().UnixNano() / 100000
}

func deleteExpireSession(sid string) {
	sessionMap.Delete(sid)
	dbops.ReleaseSession(sid)
}

func LoadSessionsFromDB() {
	r, err := dbops.RetieveSessions()
	if err != nil {
		return
	}
	r.Range(func(key, value interface{}) bool {
		ss := value.(*entity.SimpleSession)
		sessionMap.Store(key, ss)
		return true
	})
}

func GenerateNewSessionId(un string) string {
	id := uuid.NewV4().String()
	ctime := time.Now().UnixNano() / int64(time.Millisecond)
	ttl := ctime + 30*60*1000 // serverside session valid time
	ss := &entity.SimpleSession{UserName: un, TTL: ttl}
	sessionMap.Store(id, ss)
	dbops.InsertSession(id, ss.TTL, ss.UserName)
	return id
}
func IsExpireSession(sid string) (string, bool) {
	ss, ok := sessionMap.Load(sid)
	if ok {
		ct := nowInMilli()
		if ss.(*entity.SimpleSession).TTL < ct {
			deleteExpireSession(sid)
			return "", true
		}
		return ss.(*entity.SimpleSession).UserName, false
	}
	return "", true

}
