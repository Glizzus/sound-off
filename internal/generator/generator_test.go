package generator_test

import (
	"regexp"
	"sync"
	"testing"

	"github.com/glizzus/sound-off/internal/generator"
)

func TestUUIDV4Generator_Next_Concurrent(t *testing.T) {
	regex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	gen := generator.UUIDV4Generator{}

	var mu sync.Mutex
	seen := make(map[string]struct{})

	total := 100000
	concurrency := 10
	batchSize := total / concurrency

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for range concurrency {
		go func() {
			defer wg.Done()
			for range batchSize {
				id, err := gen.Next()
				if err != nil {
					t.Error("expected no error, got:", err)
					return
				}
				mu.Lock()
				if _, ok := seen[id]; ok {
					mu.Unlock()
					t.Errorf("expected a unique ID, got duplicate: %s", id)
					return
				}
				seen[id] = struct{}{}
				mu.Unlock()

				if !regex.MatchString(id) {
					t.Errorf("expected valid UUID format, got %s", id)
					return
				}
			}
		}()
	}

	wg.Wait()
}
