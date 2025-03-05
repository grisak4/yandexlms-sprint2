package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var testRouter *gin.Engine
var testMu sync.Mutex

func init() {
	gin.SetMode(gin.TestMode)
	testRouter = setupRouter()
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.POST("/api/v1/calculate", AddExpression)
	r.GET("/api/v1/expressions", GetExpressions)
	r.GET("/api/v1/expressions/:id", GetExpressionByID)
	r.GET("/internal/task", GetTask)
	r.POST("/internal/task", SubmitTaskResult)
	return r
}

func TestAddExpression(t *testing.T) {
	testMu.Lock()
	defer testMu.Unlock()

	payload := `{"expression": "2 + 2"}`

	req, _ := http.NewRequest("POST", "/api/v1/calculate", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Nil(t, err)
	assert.Contains(t, resp, "id")
}

func TestGetExpressions(t *testing.T) {
	testMu.Lock()
	defer testMu.Unlock()

	req, _ := http.NewRequest("GET", "/api/v1/expressions", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Nil(t, err)
	assert.Contains(t, resp, "expressions")
}

func TestGetExpressionByID(t *testing.T) {
	testMu.Lock()
	defer testMu.Unlock()

	// Добавляем выражение
	payload := `{"expression": "3 + 5"}`
	req, _ := http.NewRequest("POST", "/api/v1/calculate", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Nil(t, err)
	exprID := int(resp["id"].(float64))

	// Запрашиваем его по ID
	req, _ = http.NewRequest("GET", "/api/v1/expressions/"+strconv.Itoa(exprID), nil)
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// 🔹 Тест на получение задания
func TestGetTask(t *testing.T) {
	testMu.Lock()
	defer testMu.Unlock()

	req, _ := http.NewRequest("GET", "/internal/task", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	// Может быть 404, если задач нет
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound)

	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Nil(t, err)
		assert.Contains(t, resp, "task")
	}
}

// 🔹 Тест на отправку результата задачи
func TestSubmitTaskResult(t *testing.T) {
	testMu.Lock()
	defer testMu.Unlock()

	// Получаем задачу
	req, _ := http.NewRequest("GET", "/internal/task", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Skip("No tasks available to test submitTaskResult")
	}

	var taskResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &taskResp)
	assert.Nil(t, err)

	task := taskResp["task"].(map[string]interface{})
	taskID := int(task["id"].(float64))

	// Отправляем результат
	payload := map[string]interface{}{
		"id":     taskID,
		"result": 4,
	}
	body, _ := json.Marshal(payload)

	req, _ = http.NewRequest("POST", "/internal/task", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Nil(t, err)
	assert.Contains(t, resp, "message")
}
