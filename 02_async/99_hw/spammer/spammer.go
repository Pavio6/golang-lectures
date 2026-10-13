package main

import (
	"fmt"
	"sort"
	"sync"
)

func RunPipeline(cmds ...cmd) {
	wg := sync.WaitGroup{}
	in := make(chan interface{})
	close(in)

	for _, command := range cmds {
		out := make(chan interface{})
		wg.Add(1)
		go func(command cmd, in, out chan interface{}) {
			defer wg.Done()
			defer close(out)
			command(in, out)
		}(command, in, out)
		in = out
	}

	wg.Wait()
}

func SelectUsers(in, out chan interface{}) {
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	seen := make(map[uint64]struct{})

	for raw := range in {
		email := raw.(string)
		wg.Add(1)
		go func(email string) {
			defer wg.Done()

			user := GetUser(email)

			mu.Lock()
			if _, ok := seen[user.ID]; ok {
				mu.Unlock()
				return
			}
			seen[user.ID] = struct{}{}
			mu.Unlock()

			out <- user
		}(email)
	}

	wg.Wait()
}

func SelectMessages(in, out chan interface{}) {
	wg := sync.WaitGroup{}
	batch := make([]User, 0, GetMessagesMaxUsersBatch)

	flush := func(users []User) {
		if len(users) == 0 {
			return
		}
		copied := append([]User(nil), users...)
		wg.Add(1)
		go func() {
			defer wg.Done()

			msgs, err := GetMessages(copied...)
			if err != nil {
				return
			}
			for _, id := range msgs {
				out <- id
			}
		}()
	}

	for raw := range in {
		batch = append(batch, raw.(User))
		if len(batch) == GetMessagesMaxUsersBatch {
			flush(batch)
			batch = batch[:0]
		}
	}
	flush(batch)
	wg.Wait()
}

func CheckSpam(in, out chan interface{}) {
	wg := sync.WaitGroup{}
	quota := make(chan struct{}, HasSpamMaxAsyncRequests)

	for raw := range in {
		id := raw.(MsgID)
		wg.Add(1)
		quota <- struct{}{}
		go func(id MsgID) {
			defer wg.Done()
			defer func() { <-quota }()

			hasSpam, err := HasSpam(id)
			if err != nil {
				return
			}
			out <- MsgData{ID: id, HasSpam: hasSpam}
		}(id)
	}

	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	results := make([]MsgData, 0)
	for raw := range in {
		results = append(results, raw.(MsgData))
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].HasSpam != results[j].HasSpam {
			return results[i].HasSpam
		}
		return results[i].ID < results[j].ID
	})

	for _, msg := range results {
		out <- fmt.Sprintf("%t %d", msg.HasSpam, msg.ID)
	}
}
