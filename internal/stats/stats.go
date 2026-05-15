package stats

import "sync/atomic"

type Stats struct {
	totalRequests int64
	success       int64
	errors        int64
	poolHits      int64
	poolMisses    int64
	poolPuts      int64
}

type SnapshotData struct {
	TotalRequests int64 `json:"total_requests"`
	Success       int64 `json:"success"`
	Errors        int64 `json:"errors"`
	PoolHits      int64 `json:"pool_hits"`
	PoolMisses    int64 `json:"pool_misses"`
	PoolPuts      int64 `json:"pool_puts"`
}

func New() *Stats {
	return &Stats{}
}

func (s *Stats) RecordRequest()  { atomic.AddInt64(&s.totalRequests, 1) }
func (s *Stats) RecordSuccess()  { atomic.AddInt64(&s.success, 1) }
func (s *Stats) RecordError()    { atomic.AddInt64(&s.errors, 1) }
func (s *Stats) RecordPoolHit()  { atomic.AddInt64(&s.poolHits, 1) }
func (s *Stats) RecordPoolMiss() { atomic.AddInt64(&s.poolMisses, 1) }
func (s *Stats) RecordPoolPut()  { atomic.AddInt64(&s.poolPuts, 1) }

func (s *Stats) Snapshot() SnapshotData {
	return SnapshotData{
		TotalRequests: atomic.LoadInt64(&s.totalRequests),
		Success:       atomic.LoadInt64(&s.success),
		Errors:        atomic.LoadInt64(&s.errors),
		PoolHits:      atomic.LoadInt64(&s.poolHits),
		PoolMisses:    atomic.LoadInt64(&s.poolMisses),
		PoolPuts:      atomic.LoadInt64(&s.poolPuts),
	}
}
