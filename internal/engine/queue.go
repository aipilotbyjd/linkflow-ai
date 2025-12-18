// Package engine provides task queue implementations
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TaskQueue defines the interface for task queues
type TaskQueue interface {
	// Enqueue adds a task to the queue
	Enqueue(ctx context.Context, task *Task) error
	
	// Dequeue removes and returns the next task
	Dequeue(ctx context.Context) (*Task, error)
	
	// Peek returns the next task without removing it
	Peek(ctx context.Context) (*Task, error)
	
	// Ack acknowledges a task as completed
	Ack(ctx context.Context, taskID string) error
	
	// Nack returns a task to the queue (failed processing)
	Nack(ctx context.Context, taskID string) error
	
	// Len returns the number of tasks in the queue
	Len(ctx context.Context) (int64, error)
	
	// Close closes the queue
	Close() error
}

// InMemoryQueue implements an in-memory task queue
type InMemoryQueue struct {
	tasks      []*Task
	processing map[string]*Task
	mu         sync.RWMutex
	cond       *sync.Cond
	closed     bool
}

// NewInMemoryQueue creates a new in-memory queue
func NewInMemoryQueue() *InMemoryQueue {
	q := &InMemoryQueue{
		tasks:      make([]*Task, 0),
		processing: make(map[string]*Task),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Enqueue adds a task to the queue
func (q *InMemoryQueue) Enqueue(ctx context.Context, task *Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	task.CreatedAt = time.Now()

	// Insert based on priority (higher priority first)
	inserted := false
	for i, t := range q.tasks {
		if task.Priority > t.Priority {
			q.tasks = append(q.tasks[:i], append([]*Task{task}, q.tasks[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		q.tasks = append(q.tasks, task)
	}

	q.cond.Signal()
	return nil
}

// Dequeue removes and returns the next task
func (q *InMemoryQueue) Dequeue(ctx context.Context) (*Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.tasks) == 0 && !q.closed {
		q.cond.Wait()
	}

	if q.closed && len(q.tasks) == 0 {
		return nil, fmt.Errorf("queue is closed")
	}

	task := q.tasks[0]
	q.tasks = q.tasks[1:]
	q.processing[task.ID] = task
	
	now := time.Now()
	task.StartedAt = &now

	return task, nil
}

// Peek returns the next task without removing it
func (q *InMemoryQueue) Peek(ctx context.Context) (*Task, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if len(q.tasks) == 0 {
		return nil, nil
	}

	return q.tasks[0], nil
}

// Ack acknowledges a task as completed
func (q *InMemoryQueue) Ack(ctx context.Context, taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	delete(q.processing, taskID)
	return nil
}

// Nack returns a task to the queue
func (q *InMemoryQueue) Nack(ctx context.Context, taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, exists := q.processing[taskID]
	if !exists {
		return fmt.Errorf("task %s not found in processing", taskID)
	}

	delete(q.processing, taskID)
	task.RetryCount++
	
	// Add back to queue with lower priority
	task.Priority--
	q.tasks = append(q.tasks, task)

	return nil
}

// Len returns the number of tasks in the queue
func (q *InMemoryQueue) Len(ctx context.Context) (int64, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return int64(len(q.tasks)), nil
}

// Close closes the queue
func (q *InMemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.closed = true
	q.cond.Broadcast()
	return nil
}

// RedisQueue implements a Redis-based task queue for distributed execution
type RedisQueue struct {
	client        *redis.Client
	queueKey      string
	processingKey string
	deadLetterKey string
	visTimeout    time.Duration
}

// RedisQueueConfig holds Redis queue configuration
type RedisQueueConfig struct {
	Addr              string
	Password          string
	DB                int
	QueueName         string
	VisibilityTimeout time.Duration
}

// NewRedisQueue creates a new Redis-based queue
func NewRedisQueue(config *RedisQueueConfig) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	queueName := config.QueueName
	if queueName == "" {
		queueName = "linkflow:tasks"
	}

	visTimeout := config.VisibilityTimeout
	if visTimeout == 0 {
		visTimeout = 5 * time.Minute
	}

	return &RedisQueue{
		client:        client,
		queueKey:      queueName,
		processingKey: queueName + ":processing",
		deadLetterKey: queueName + ":deadletter",
		visTimeout:    visTimeout,
	}, nil
}

// Enqueue adds a task to the queue
func (q *RedisQueue) Enqueue(ctx context.Context, task *Task) error {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	task.CreatedAt = time.Now()

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Use ZADD with priority as score (higher priority = lower score for ZRANGEBYSCORE)
	score := float64(time.Now().UnixNano()) - float64(task.Priority*1000000000)
	
	return q.client.ZAdd(ctx, q.queueKey, redis.Z{
		Score:  score,
		Member: data,
	}).Err()
}

// Dequeue removes and returns the next task
func (q *RedisQueue) Dequeue(ctx context.Context) (*Task, error) {
	// Use ZPOPMIN to get highest priority (lowest score) task
	results, err := q.client.ZPopMin(ctx, q.queueKey, 1).Result()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	data := results[0].Member.(string)
	
	var task Task
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	now := time.Now()
	task.StartedAt = &now

	// Add to processing set with expiry
	processingData, _ := json.Marshal(task)
	q.client.HSet(ctx, q.processingKey, task.ID, processingData)
	q.client.Expire(ctx, q.processingKey+":"+task.ID, q.visTimeout)

	return &task, nil
}

// Peek returns the next task without removing it
func (q *RedisQueue) Peek(ctx context.Context) (*Task, error) {
	results, err := q.client.ZRange(ctx, q.queueKey, 0, 0).Result()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	var task Task
	if err := json.Unmarshal([]byte(results[0]), &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// Ack acknowledges a task as completed
func (q *RedisQueue) Ack(ctx context.Context, taskID string) error {
	return q.client.HDel(ctx, q.processingKey, taskID).Err()
}

// Nack returns a task to the queue
func (q *RedisQueue) Nack(ctx context.Context, taskID string) error {
	// Get task from processing
	data, err := q.client.HGet(ctx, q.processingKey, taskID).Result()
	if err != nil {
		return err
	}

	var task Task
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		return err
	}

	// Remove from processing
	q.client.HDel(ctx, q.processingKey, taskID)

	// Check retry count
	task.RetryCount++
	if task.RetryCount > task.MaxRetries {
		// Move to dead letter queue
		dlData, _ := json.Marshal(task)
		return q.client.LPush(ctx, q.deadLetterKey, dlData).Err()
	}

	// Re-enqueue with delay
	task.Priority--
	return q.Enqueue(ctx, &task)
}

// Len returns the number of tasks in the queue
func (q *RedisQueue) Len(ctx context.Context) (int64, error) {
	return q.client.ZCard(ctx, q.queueKey).Result()
}

// Close closes the queue
func (q *RedisQueue) Close() error {
	return q.client.Close()
}

// GetProcessingCount returns number of tasks being processed
func (q *RedisQueue) GetProcessingCount(ctx context.Context) (int64, error) {
	return q.client.HLen(ctx, q.processingKey).Result()
}

// GetDeadLetterCount returns number of tasks in dead letter queue
func (q *RedisQueue) GetDeadLetterCount(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.deadLetterKey).Result()
}

// ReprocessDeadLetter moves tasks from dead letter queue back to main queue
func (q *RedisQueue) ReprocessDeadLetter(ctx context.Context, count int) (int, error) {
	processed := 0
	
	for i := 0; i < count; i++ {
		data, err := q.client.RPop(ctx, q.deadLetterKey).Result()
		if err == redis.Nil {
			break
		}
		if err != nil {
			return processed, err
		}

		var task Task
		if err := json.Unmarshal([]byte(data), &task); err != nil {
			continue
		}

		task.RetryCount = 0
		if err := q.Enqueue(ctx, &task); err != nil {
			// Put back in dead letter
			q.client.LPush(ctx, q.deadLetterKey, data)
			return processed, err
		}

		processed++
	}

	return processed, nil
}

// PriorityQueue wraps a queue with priority handling
type PriorityQueue struct {
	queues map[int]TaskQueue
	levels []int
	mu     sync.RWMutex
}

// NewPriorityQueue creates a priority queue with multiple levels
func NewPriorityQueue(levels []int) *PriorityQueue {
	pq := &PriorityQueue{
		queues: make(map[int]TaskQueue),
		levels: levels,
	}

	for _, level := range levels {
		pq.queues[level] = NewInMemoryQueue()
	}

	return pq
}

// Enqueue adds a task to the appropriate priority queue
func (pq *PriorityQueue) Enqueue(ctx context.Context, task *Task) error {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	// Find the closest priority level
	targetLevel := pq.levels[0]
	for _, level := range pq.levels {
		if task.Priority >= level {
			targetLevel = level
		}
	}

	queue, exists := pq.queues[targetLevel]
	if !exists {
		return fmt.Errorf("no queue for priority level %d", targetLevel)
	}

	return queue.Enqueue(ctx, task)
}

// Dequeue gets the next task from highest priority queue
func (pq *PriorityQueue) Dequeue(ctx context.Context) (*Task, error) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	// Check queues in priority order (highest first)
	for i := len(pq.levels) - 1; i >= 0; i-- {
		queue := pq.queues[pq.levels[i]]
		task, err := queue.Peek(ctx)
		if err != nil {
			continue
		}
		if task != nil {
			return queue.Dequeue(ctx)
		}
	}

	// If all queues empty, wait on lowest priority queue
	return pq.queues[pq.levels[0]].Dequeue(ctx)
}

// Peek returns the next task from highest priority queue
func (pq *PriorityQueue) Peek(ctx context.Context) (*Task, error) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	for i := len(pq.levels) - 1; i >= 0; i-- {
		queue := pq.queues[pq.levels[i]]
		task, err := queue.Peek(ctx)
		if err != nil {
			continue
		}
		if task != nil {
			return task, nil
		}
	}

	return nil, nil
}

// Ack acknowledges a task
func (pq *PriorityQueue) Ack(ctx context.Context, taskID string) error {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	for _, queue := range pq.queues {
		queue.Ack(ctx, taskID)
	}
	return nil
}

// Nack returns a task to the queue
func (pq *PriorityQueue) Nack(ctx context.Context, taskID string) error {
	// Would need to track which queue the task came from
	return nil
}

// Len returns total tasks across all queues
func (pq *PriorityQueue) Len(ctx context.Context) (int64, error) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	var total int64
	for _, queue := range pq.queues {
		count, err := queue.Len(ctx)
		if err != nil {
			continue
		}
		total += count
	}

	return total, nil
}

// Close closes all queues
func (pq *PriorityQueue) Close() error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	for _, queue := range pq.queues {
		queue.Close()
	}
	return nil
}
