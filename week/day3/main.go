package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// 定义job
type Job struct {
	Url      string
	FileName string
}

type Result struct {
	Job      Job
	Size     int
	Error    error
	WorkerID int
}

// 下载方法

func download(ctx context.Context, client *http.Client, job Job) (int, error) {

	// 发请求
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, job.Url, nil)
	if err != nil {
		return 0, err
	}

	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}

	return len(body), nil

}

func worker(ctx context.Context, id int, client *http.Client, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) { // 定义 worker 函数，从任务队列读取任务并把结果写入结果队列。
	defer wg.Done()
	for {

		select {

		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			size, err := download(ctx, client, job)
			result := &Result{Job: job, Size: size, Error: err, WorkerID: id}
			select {
			case <-ctx.Done():
				return
			case results <- *result:

			}
		}
	}
}

func main() {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("stable image bytes"))

	}))
	defer server.Close()
	client := server.Client()
	jobs := make(chan Job, 10)
	results := make(chan Result, 10)
	go func() {
		defer close(jobs)
		for i := 0; i < 20; i++ {
			job := Job{Url: fmt.Sprintf("%s/image/%d", server.URL, i), FileName: fmt.Sprintf("img_%d.jpg", i)} // 构造当前序号对应的下载任务。
			select {
			case <-ctx.Done():
				return
			case jobs <- job:
			}
		}
	}()
	var wg sync.WaitGroup
	numWorkers := 3
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(ctx, i, client, jobs, results, &wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()
	successCount := 0 // 初始化成功任务计数。
	failCount := 0

	for result := range results {
		if result.Error != nil {
			failCount++
			fmt.Printf("[worker-%d] FAIL %s: %v\n", result.WorkerID, result.Job.FileName, result.Error) // 打印带 worker 标签的失败日志。
			continue
		}
		successCount++
		fmt.Printf("[worker-%d] OK %s: %d bytes\n", result.WorkerID, result.Job.FileName, result.Size) // 打印带 worker 标签的成功日志。
	}
	fmt.Printf("Total: %d success, %d fail\n", successCount, failCount) // 打印最终成功和失败总数。

}
