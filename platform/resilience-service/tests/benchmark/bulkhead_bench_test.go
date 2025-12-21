package benchmark

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/libs/go/resilience/bulkhead"
)

func BenchmarkBulkhead_Acquire_Release(b *testing.B) {
	bh := bulkhead.New(bulkhead.Config{
		Name:          "bench",
		MaxConcurrent: 1000,
		MaxQueue:      1000,
		QueueTimeout:  time.Second,
	})

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := bh.Acquire(ctx); err == nil {
				bh.Release()
			}
		}
	})
}

func BenchmarkBulkhead_Acquire_NoQueue(b *testing.B) {
	bh := bulkhead.New(bulkhead.Config{
		Name:          "bench",
		MaxConcurrent: 10000,
		MaxQueue:      0,
		QueueTimeout:  time.Second,
	})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := bh.Acquire(ctx); err == nil {
			bh.Release()
		}
	}
}

func BenchmarkBulkhead_GetMetrics(b *testing.B) {
	bh := bulkhead.New(bulkhead.Config{
		Name:          "bench",
		MaxConcurrent: 100,
		MaxQueue:      100,
		QueueTimeout:  time.Second,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = bh.GetMetrics()
		}
	})
}

func BenchmarkBulkhead_HighContention(b *testing.B) {
	bh := bulkhead.New(bulkhead.Config{
		Name:          "bench",
		MaxConcurrent: 10,
		MaxQueue:      100,
		QueueTimeout:  time.Second,
	})

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := bh.Acquire(ctx); err == nil {
				// Simulate work
				time.Sleep(time.Microsecond)
				bh.Release()
			}
		}
	})
}

func BenchmarkBulkheadManager_Acquire(b *testing.B) {
	manager := bulkhead.NewManager(resilience.BulkheadConfig{
		MaxConcurrent: 100,
		MaxQueue:      100,
		QueueTimeout:  time.Second,
	}, nil)

	ctx := context.Background()
	partitions := []string{"partition-1", "partition-2", "partition-3", "partition-4"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			partition := partitions[i%len(partitions)]
			bh := manager.GetBulkhead(partition)
			if err := bh.Acquire(ctx); err == nil {
				bh.Release()
			}
			i++
		}
	})
}

func BenchmarkBulkheadManager_GetAllMetrics(b *testing.B) {
	manager := bulkhead.NewManager(resilience.BulkheadConfig{
		MaxConcurrent: 100,
		MaxQueue:      100,
		QueueTimeout:  time.Second,
	}, nil)

	ctx := context.Background()

	// Create some partitions
	for i := 0; i < 10; i++ {
		bh := manager.GetBulkhead("partition-" + string(rune('0'+i)))
		bh.Acquire(ctx)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GetAllMetrics()
	}
}

func BenchmarkBulkhead_Concurrent_100(b *testing.B) {
	benchmarkConcurrent(b, 100)
}

func BenchmarkBulkhead_Concurrent_1000(b *testing.B) {
	benchmarkConcurrent(b, 1000)
}

func benchmarkConcurrent(b *testing.B, concurrency int) {
	bh := bulkhead.New(bulkhead.Config{
		Name:          "bench",
		MaxConcurrent: concurrency,
		MaxQueue:      concurrency,
		QueueTimeout:  time.Second,
	})

	ctx := context.Background()

	b.ResetTimer()

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(concurrency)
		for j := 0; j < concurrency; j++ {
			go func() {
				defer wg.Done()
				if err := bh.Acquire(ctx); err == nil {
					bh.Release()
				}
			}()
		}
		wg.Wait()
	}
}
