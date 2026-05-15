package pool

import "sync"

var bufPools sync.Map

func GetBuf(size int) []byte {
	poolI, _ := bufPools.LoadOrStore(size, &sync.Pool{
		New: func() any {
			return make([]byte, size)
		},
	})
	pool := poolI.(*sync.Pool)
	buf := pool.Get().([]byte)
	return buf[:size]
}

func PutBuf(size int, buf []byte) {
	poolI, ok := bufPools.Load(size)
	if !ok {
		return
	}
	pool := poolI.(*sync.Pool)
	pool.Put(buf)
}
