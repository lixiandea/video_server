package entity

type UserCredential struct {
	UserName string `json:"user_name"`
	Pwd      string `json:"password"`
}

type NewVideo struct {
	AuthorId int    `json:"author_id"`
	Name     string `json:"name"`
}

type User struct {
	Id        int    `json:"id"`
	LoginName string `json:"login_name"`
	Pwd       string `json:"pwd"`
}

type UserInfo struct {
	Id int `json:"id"`
}

type VideoInfo struct {
	Id          string `json:"id"`
	AuthorId    int    `json:"author_id"`
	Name        string `json:"name"`
	DisplayTime string `json:"display_time"`
	CreateTime  string `json:"create_time"`
}

type VideosInfo struct {
	Videos []*VideoInfo `json:"videos"`
}
type Comment struct {
	Id       string `json:"id"`
	AuthorId int    `json:"author_id"`
	VideoId  string `json:"video_id"`
	Content  string `json:"content"`
	Ctime    string `json:"ctime"`
}

type Comments struct {
	Comments []*Comment `json:"comments"`
}
type SimpleSession struct {
	UserName string `json:"user_name"`
	TTL      int64  `json:"ttl"`
}
