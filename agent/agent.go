package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	api = "http://localhost:8081/internal/task"
)

type Task struct {
	ID            int     `json:"id"`
	Arg1          float64 `json:"arg1"`
	Arg2          float64 `json:"arg2"`
	Operation     string  `json:"operation"`
	OperationTime int     `json:"operation_time"`
}

func compute(task Task) float64 {
	time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)

	switch task.Operation {
	case "+":
		return task.Arg1 + task.Arg2
	case "-":
		return task.Arg1 - task.Arg2
	case "*":
		return task.Arg1 * task.Arg2
	case "/":
		if task.Arg2 == 0 {
			return math.NaN()
		}
		return task.Arg1 / task.Arg2
	}
	return 0
}

func sendResult(task Task, result float64) {
	body, _ := json.Marshal(map[string]interface{}{
		"id":     task.ID,
		"result": result,
	})

	_, err := http.Post(api, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Failed to send result: %v", err)
	}
}

func worker() {
	for {
		resp, err := http.Get(api)
		if err != nil || resp.StatusCode != 200 {
			time.Sleep(2 * time.Second)
			continue
		}

		var res struct {
			Task Task `json:"task"`
		}
		json.NewDecoder(resp.Body).Decode(&res)

		result := compute(res.Task)
		sendResult(res.Task, result)
	}
}

func main() {
	workers, _ := strconv.Atoi(os.Getenv("COMPUTING_POWER"))
	if workers == 0 {
		workers = 2
	}

	for i := 0; i < workers; i++ {
		go worker()
	}

	select {}
}
