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
	Id        int
	LoginName string
	Pwd       string
}

type UserInfo struct {
	Id int `json:"id"`
}

type VideoInfo struct {
	Id          string
	AuthorId    int
	Name        string
	DisplayTime string
	CreateTime  string
}

type VideosInfo struct {
	Videos []*VideoInfo
}
type Comment struct {
	Id       string
	AuthorId int
	VideoId  string
	Content  string
	Ctime    string
}

type Comments struct {
	Comments []*Comment
}
type SimpleSession struct {
	UserName string
	TTL      int64
}
