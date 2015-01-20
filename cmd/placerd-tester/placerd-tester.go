package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/pkg/queue"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)

	actions := []string{"start", "stop", "create", "delete"}
	data := `{"action":"` + actions[rand.Intn(len(actions))] + `","guest":"` + uuid.New() + `"}`
	q := queue.Connect(etcd.NewClient([]string{"http://localhost:4001"}), "/queue")

	if os.Getenv("RAND_SEED") == "" {
		rand.Seed(time.Now().UnixNano())
	} else {
		seed := int64(0)
		n, err := fmt.Sscan(os.Getenv("RAND_SEED"), &seed)
		if err != nil {
			panic(err)
		} else if n != 1 {
			panic("incorrect number of args scanned")
		}
		rand.Seed(seed)
	}
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
