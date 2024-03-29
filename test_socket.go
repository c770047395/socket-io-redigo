package main

import (
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	socketio "github.com/googollee/go-socket.io"
	"log"
	"net/http"
)


var server *socketio.Server

type Msg struct {
	Room string `json:"room"`
	Content string `json:"content"`
}

func connRedis(){
	c, err := redis.Dial("tcp", "47.96.128.98:6379")
	if err != nil {
		log.Fatalln(err)
	}
	defer c.Close()
	_, err = c.Do("AUTH",  "123456")
	if err != nil {
		fmt.Println("认证失败:", err)
	}

	fmt.Println("接收消息....")

	subChan(c,"chan1")
}

func subChan(c redis.Conn,channame string){
	psc := redis.PubSubConn{c}
	_ = psc.Subscribe(channame)
	for{
		switch v := psc.Receive().(type) {
		case redis.Message:
			fmt.Printf("%s: message: %s\n", v.Channel, v.Data)
			var data Msg
			err := json.Unmarshal(v.Data,&data)
			if err != nil{
				fmt.Println("json解析失败:",err)
				return
			}
			server.BroadcastToRoom(data.Room,"reply",data.Content)
		case redis.Subscription:
			fmt.Printf("%s: %s %d\n", v.Channel, v.Kind, v.Count)
		case error:
			fmt.Println(v)
			return
		}
	}
}

func serverInit(){
	var err error
	server, err = socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
}

func socketServer(){

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())
		return nil
	})
	server.OnEvent("/","joinRoom", func(s socketio.Conn,msg string) {
		s.Emit("reply","welcome to 1-room")
		s.Join("1")
	})
	server.OnEvent("/", "noticecp", func(s socketio.Conn, msg string) {
		fmt.Println("notice:", msg)
		server.BroadcastToRoom("1","reply",s.ID()+":"+msg)
	})

	server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
		s.SetContext(msg)
		return "recv " + msg
	})

	server.OnEvent("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		s.Close()
		return last
	})
	server.OnError("/", func(e error) {
		fmt.Println("meet error:", e)
	})
	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		fmt.Println("closed", msg)
	})
	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost:8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))

}


func main() {
	serverInit()
	go socketServer()
	connRedis()
}