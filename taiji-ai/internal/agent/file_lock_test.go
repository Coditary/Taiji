package agent

import (
	"sync"
	"testing"
	"time"
)

func TestLockFilePathsSerializesSameFile(t *testing.T) {
	path := t.TempDir() + "/same.txt"
	var active int
	var peak int
	var mu sync.Mutex

	run := func() {
		unlock := lockFilePaths([]string{path})
		mu.Lock()
		active++
		if active > peak {
			peak = active
		}
		mu.Unlock()
		time.Sleep(15 * time.Millisecond)
		mu.Lock()
		active--
		mu.Unlock()
		unlock()
	}

	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			run()
		}()
	}
	wg.Wait()
	if peak > 1 {
		t.Fatalf("expected serialized access, peak concurrent=%d", peak)
	}
}
