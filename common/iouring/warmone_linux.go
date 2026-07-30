//go:build linux

package iouring

import (
	"os"
	"runtime"
	"sync"

	"github.com/erigontech/erigon/common/dbg"
	"github.com/erigontech/erigon/common/log/v3"
)

// Pool of small rings for per-access warming: each goroutine borrows a ring,
// does a single-read submit+wait (which releases the P via io_uring_enter),
// and returns it. Sized to the expected number of concurrent cold readers.
const warmBufSize = 32 * 1024

// poolSize bounds concurrent warms at ~2*GOMAXPROCS: a blocking fault holds a P,
// and an io_uring read frees it for one more, so beyond that rings sit idle. The
// cap also keeps the per-process io_uring memory footprint small (each instance
// pins kernel memory; hundreds of rings can exhaust it).
func poolSize() int {
	if n := dbg.EnvInt("RESIDENCY_IOURING_RINGS", 0); n > 0 {
		return n
	}
	return min(2*runtime.GOMAXPROCS(0), 64)
}

type pooledRing struct {
	r   *Ring
	buf []byte
	// reusable single-read argument slots, so a warm allocates nothing per call
	offs [1]int64
	lens [1]int
	bufs [1][]byte
}

// newRing builds a pooled ring. io_uring is required — there is no fallback — so
// a setup failure exits the process rather than silently degrading to blocking reads.
func newRing() *pooledRing {
	r, err := New(8)
	if err != nil {
		log.Crit("[IOURING] io_uring_setup failed; kernel does not support it or it is blocked (seccomp)", "err", err)
		os.Exit(1)
	}
	return &pooledRing{r: r, buf: make([]byte, warmBufSize)}
}

var (
	ringPool     chan *pooledRing
	ringPoolOnce sync.Once
)

func initPool() {
	n := poolSize()
	ringPool = make(chan *pooledRing, n)
	for range n {
		ringPool <- newRing()
	}
}

// WarmOne reads [off, off+length) via a pooled io_uring ring to populate the
// page cache. It blocks for a free ring when the pool is drained (parking the
// goroutine, freeing its P). There is no fallback: io_uring must be available,
// and the process exits the first time a warm is needed if setup fails.
func WarmOne(fd int, off int64, length int) {
	ringPoolOnce.Do(initPool)
	if length > warmBufSize {
		length = warmBufSize
	}
	pr := <-ringPool
	pr.offs[0], pr.lens[0], pr.bufs[0] = off, length, pr.buf
	if err := pr.r.BatchReadWarm(fd, pr.offs[:], pr.lens[:], pr.bufs[:]); err != nil {
		// A ring that errored may still have completions pending in the kernel;
		// reusing it would miscount the next read's reap. Discard and refill.
		pr.r.Close()
		ringPool <- newRing()
		return
	}
	ringPool <- pr
}

// Available reports whether io_uring can be set up in this process (kernel
// support, not blocked by seccomp). For tests that must skip on such hosts.
func Available() bool {
	r, err := New(1)
	if err != nil {
		return false
	}
	r.Close()
	return true
}
