package pool

type BytePool struct {
	ch        chan []byte
	classSize int
}

func NewBytePool(capacity int, classSize int) *BytePool {
	return &BytePool{
		ch:        make(chan []byte, capacity),
		classSize: classSize,
	}
}

func (p *BytePool) Get(size int) []byte {
	select {
	case buf := <-p.ch:
		if cap(buf) >= size {
			return buf[:size]
		}
	default:
	}
	return make([]byte, size)
}

func (p *BytePool) Put(buf []byte) {
	buf = buf[:cap(buf)]
	select {
	case p.ch <- buf:
	default:
	}
}

var bytePools = [4]*BytePool{
	NewBytePool(64, 1024),
	NewBytePool(64, 4096),
	NewBytePool(64, 16384),
	NewBytePool(64, 65536),
}

func classifySize(size int) int {
	switch {
	case size <= 1024:
		return 0
	case size <= 4096:
		return 1
	case size <= 16384:
		return 2
	default:
		return 3
	}
}

func GetBufClass(size int) []byte {
	idx := classifySize(size)
	return bytePools[idx].Get(size)
}

func PutBufClass(size int, buf []byte) {
	idx := classifySize(size)
	bytePools[idx].Put(buf)
}
