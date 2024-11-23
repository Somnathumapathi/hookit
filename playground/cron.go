// package main

// import (
// 	"sync"
// 	"time"

// 	"gofr.dev/pkg/gofr"
// )

// var (
// 	n  = 0
// 	mu sync.RWMutex
// )

// const duration = 3

// func main() {
// 	app := gofr.New()

// 	app.AddCronJob("* * * * * *", "counter", count)
// 	time.Sleep(duration * time.Second)
// }

// func count(c *gofr.Context) {
// 	mu.Lock()
// 	defer mu.Unlock()

// 	n++

// 	c.Log("Count:", n)
// }