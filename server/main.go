package main

import (
	"dvecdb/api"
	"dvecdb/kvstore"
	"dvecdb/replication/raft"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

var kv *kvstore.KvStore

// Inside main.go
const (
	etcdEndpoints = "localhost:2379" // This is correct for your Docker setup
	servicePrefix = "/services/my-app/"
	ttl           = 5
)

func main() {

	app := echo.New()

	kv = kvstore.NewKvStore()

	group := app.Group("/data")
	api.SetApiRoutes(group)

	ch := make(chan bool)
	go func() {
		for {
			ticker := time.NewTicker(time.Minute * time.Duration(1))
			select {
			case <-ticker.C:
				go kvstore.SnapShot(ch)
			case <-ch:
				fmt.Println("snapshot taken")
			}
		}
	}()

	// List of known peers (including itself)
	peers := []string{
		"localhost:5101",
		"localhost:5102",
		"localhost:5103",
	}

	node1 := raft.NewNode("1", "localhost:5101")
	node2 := raft.NewNode("2", "localhost:5102")
	node3 := raft.NewNode("3", "localhost:5103")

	go node2.Run(peers)
	go node3.Run(peers)
	go node1.Run(peers)

	app.Start(":3000")
	select {}
}
