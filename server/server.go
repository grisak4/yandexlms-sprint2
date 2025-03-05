package main

import (
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Expression struct {
	ID     int         `json:"id"`
	Status string      `json:"status"`
	Result interface{} `json:"result"`
	Tasks  []*Task     `json:"-"`
}

type Task struct {
	ID            int     `json:"id"`
	Arg1          float64 `json:"arg1"`
	Arg2          float64 `json:"arg2"`
	Operation     string  `json:"operation"`
	OperationTime int     `json:"operation_time"`
	Done          bool    `json:"-"`
}

var (
	expressions = make(map[int]*Expression)
	tasksQueue  []*Task
	mu          sync.Mutex
	exprID      = 1
	taskID      = 1
)

func parseExpression(expression string) []*Task {
	tokens := strings.Fields(expression)
	if len(tokens) < 3 {
		return nil
	}

	var tasks []*Task
	for i := 1; i < len(tokens)-1; i += 2 {
		arg1, _ := strconv.ParseFloat(tokens[i-1], 64)
		arg2, _ := strconv.ParseFloat(tokens[i+1], 64)
		op := tokens[i]

		tasks = append(tasks, &Task{
			ID:            taskID,
			Arg1:          arg1,
			Arg2:          arg2,
			Operation:     op,
			OperationTime: rand.Intn(5000) + 1000,
		})
		taskID++
	}
	return tasks
}

func AddExpression(c *gin.Context) {
	var req struct {
		Expression string `json:"expression"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid expression"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	tasks := parseExpression(req.Expression)
	if tasks == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid expression format"})
		return
	}

	expr := &Expression{
		ID:     exprID,
		Status: "pending",
		Tasks:  tasks,
	}
	expressions[exprID] = expr
	exprID++

	tasksQueue = append(tasksQueue, tasks...)
	c.JSON(http.StatusCreated, gin.H{"id": expr.ID})
}

func GetExpressions(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()

	var response []Expression
	for _, expr := range expressions {
		response = append(response, *expr)
	}

	c.JSON(http.StatusOK, gin.H{"expressions": response})
}

func GetExpressionByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expression not found"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	expr, exists := expressions[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expression not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"expression": expr})
}

func GetTask(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()

	if len(tasksQueue) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No tasks available"})
		return
	}

	task := tasksQueue[0]
	tasksQueue = tasksQueue[1:]

	c.JSON(http.StatusOK, gin.H{"task": task})
}

func SubmitTaskResult(c *gin.Context) {
	var result struct {
		ID     int     `json:"id"`
		Result float64 `json:"result"`
	}

	if err := c.ShouldBindJSON(&result); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid data"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for _, expr := range expressions {
		for _, task := range expr.Tasks {
			if task.ID == result.ID {
				task.Done = true
				expr.Result = result.Result

				allDone := true
				for _, t := range expr.Tasks {
					if !t.Done {
						allDone = false
						break
					}
				}
				if allDone {
					expr.Status = "done"
				}

				c.JSON(http.StatusOK, gin.H{"message": "Result saved"})
				return
			}
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
}

func main() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Разрешаем только localhost
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	r.POST("/api/v1/calculate", AddExpression)
	r.GET("/api/v1/expressions", GetExpressions)
	r.GET("/api/v1/expressions/:id", GetExpressionByID)
	r.GET("/internal/task", GetTask)
	r.POST("/internal/task", SubmitTaskResult)

	r.Run(":8081")
}
