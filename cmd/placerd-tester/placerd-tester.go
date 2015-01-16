package main

import (
	"log"
	"math/rand"
	"sync"

	"code.google.com/p/go-uuid/uuid"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/pkg/queue"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)

	actions := []string{"start", "stop", "create", "delete"}
	data := `{"action":"` + actions[rand.Intn(len(actions))] + `","guest":"` + uuid.New() + `"}`
	q := queue.Connect(etcd.NewClient([]string{"http://localhost:4001"}), "/queue")

	n := rand.Intn(100)
	wg := sync.WaitGroup{}
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			resp, err := q.Put(data)
			if err != nil {
				panic(err)
				//fmt.Println(err)
			}
			log.Println(i, "resp:", resp)
			wg.Done()
		}(i)
	}
	wg.Wait()
}
