package timeserie

import (
	"maps"
	"sync"
)

type TimeSeriesCollector struct {
	mu      sync.Mutex
	buffers map[uint32][]*GpuEventToken 
}

func NewTimeSeriesCollector() *TimeSeriesCollector {
	return &TimeSeriesCollector{
		buffers: make(map[uint32][]*GpuEventToken),
	}
}

func (tc *TimeSeriesCollector) Update(ev any) {
	token := EventToToken(ev)
	if token == nil {
		return
	}
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.buffers[token.PID] = append(tc.buffers[token.PID], token)
}

func (tc *TimeSeriesCollector) Flush() map[uint32][]*GpuEventToken {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	result := make(map[uint32][]*GpuEventToken, len(tc.buffers))
	maps.Copy(result, tc.buffers)
	tc.buffers = make(map[uint32][]*GpuEventToken) 
	return result
}

