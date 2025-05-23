// Copyright 2018 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package devpts

import (
	"gvisor.dev/gvisor/pkg/abi/linux"
	"gvisor.dev/gvisor/pkg/context"
	"gvisor.dev/gvisor/pkg/errors/linuxerr"
	"gvisor.dev/gvisor/pkg/marshal/primitive"
	"gvisor.dev/gvisor/pkg/safemem"
	"gvisor.dev/gvisor/pkg/sentry/arch"
	"gvisor.dev/gvisor/pkg/sentry/kernel"
	"gvisor.dev/gvisor/pkg/sync"
	"gvisor.dev/gvisor/pkg/usermem"
	"gvisor.dev/gvisor/pkg/waiter"
)

// waitBufMaxBytes is the maximum size of a wait buffer. It is based on
// TTYB_DEFAULT_MEM_LIMIT.
const waitBufMaxBytes = 131072

// queue represents one of the input or output queues between a pty master and
// replica. Bytes written to a queue are added to the read buffer until it is
// full, at which point they are written to the wait buffer. Bytes are
// processed (i.e. undergo termios transformations) as they are added to the
// read buffer. The read buffer is readable when its length is nonzero and
// readable is true, or when its length is zero and readable is true (EOF).
//
// +stateify savable
type queue struct {
	// mu protects everything in queue.
	mu sync.Mutex `state:"nosave"`

	// readBuf is buffer of data ready to be read when readable is true.
	// This data has been processed.
	readBuf []byte

	// waitBuf contains data that can't fit into readBuf. It is put here
	// until it can be loaded into the read buffer. waitBuf contains data
	// that hasn't been processed.
	waitBuf    [][]byte
	waitBufLen uint64

	// readable indicates whether the read buffer can be read from.  In
	// canonical mode, there can be an unterminated line in the read buffer,
	// so readable must be checked.
	readable bool

	// transform is the queue's function for transforming bytes
	// entering the queue. For example, transform might convert all '\r's
	// entering the queue to '\n's.
	transformer
}

// readReadiness returns whether q is ready to be read from.
func (q *queue) readReadiness(t *linux.KernelTermios) waiter.EventMask {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.readBuf) > 0 && q.readable {
		return waiter.ReadableEvents
	}
	return waiter.EventMask(0)
}

// writeReadiness returns whether q is ready to be written to.
func (q *queue) writeReadiness(t *linux.KernelTermios) waiter.EventMask {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.waitBufLen < waitBufMaxBytes {
		return waiter.WritableEvents
	}
	return waiter.EventMask(0)
}

// readableSize writes the number of readable bytes to userspace.
func (q *queue) readableSize(t *kernel.Task, io usermem.IO, args arch.SyscallArguments) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	size := primitive.Int32(0)
	if q.readable {
		size = primitive.Int32(len(q.readBuf))
	}

	_, err := size.CopyOut(t, args[2].Pointer())
	return err

}

// read reads from q to userspace. It returns:
//   - The number of bytes read
//   - Whether the read caused more readable data to become available (whether
//     data was pushed from the wait buffer to the read buffer).
//   - Whether any data was echoed back (need to notify readers).
//
// Preconditions: l.termiosMu must be held for reading.
func (q *queue) read(ctx context.Context, dst usermem.IOSequence, l *lineDiscipline, packet bool) (int64, bool, bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.readable {
		if l.numReplicas == 0 {
			return 0, false, false, linuxerr.EIO
		}
		return 0, false, false, linuxerr.ErrWouldBlock
	}

	// In packet mode, we write a leading data header byte, and this byte
	// should be accounted for in the return value from read.
	var pktHdrLen int
	if packet {
		// Write leading data header byte.
		var err error
		if pktHdrLen, err = dst.CopyOut(ctx, []byte{linux.TIOCPKT_DATA}); err != nil {
			return 0, false, false, err
		}
		dst = dst.DropFirst(pktHdrLen)
	}

	if dst.NumBytes() > canonMaxBytes {
		dst = dst.TakeFirst(canonMaxBytes)
	}

	n, err := dst.CopyOutFrom(ctx, safemem.ReaderFunc(func(dst safemem.BlockSeq) (uint64, error) {
		src := safemem.BlockSeqOf(safemem.BlockFromSafeSlice(q.readBuf))
		n, err := safemem.CopySeq(dst, src)
		if err != nil {
			return 0, err
		}
		q.readBuf = q.readBuf[n:]

		// If we read everything, this queue is no longer readable.
		if len(q.readBuf) == 0 {
			q.readable = false
		}

		return n, nil
	}))
	if err != nil {
		return 0, false, false, err
	}

	// Move data from the queue's wait buffer to its read buffer.
	nPushed, notifyEcho := q.pushWaitBufLocked(l)

	return int64(n) + int64(pktHdrLen), nPushed > 0, notifyEcho, nil
}

// write writes to q from userspace.
// The returned boolean indicates whether any data was echoed back.
//
// Preconditions: l.termiosMu must be held for reading.
func (q *queue) write(ctx context.Context, src usermem.IOSequence, l *lineDiscipline) (int64, bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Copy data into the wait buffer.
	n, err := src.CopyInTo(ctx, safemem.WriterFunc(func(src safemem.BlockSeq) (uint64, error) {
		copyLen := src.NumBytes()
		room := waitBufMaxBytes - q.waitBufLen
		// If out of room, return EAGAIN.
		if room == 0 && copyLen > 0 {
			return 0, linuxerr.ErrWouldBlock
		}
		// Cap the size of the wait buffer.
		if copyLen > room {
			copyLen = room
			src = src.TakeFirst64(room)
		}
		buf := make([]byte, copyLen)

		// Copy the data into the wait buffer.
		dst := safemem.BlockSeqOf(safemem.BlockFromSafeSlice(buf))
		n, err := safemem.CopySeq(dst, src)
		if err != nil {
			return 0, err
		}
		q.waitBufAppend(buf)

		return n, nil
	}))
	if err != nil {
		return 0, false, err
	}

	// Push data from the wait to the read buffer.
	_, notifyEcho := q.pushWaitBufLocked(l)

	return n, notifyEcho, nil
}

// writeBytes writes to q from b.
// The returned boolean indicates whether any data was echoed back.
//
// Preconditions: l.termiosMu must be held for reading.
func (q *queue) writeBytes(b []byte, l *lineDiscipline) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Write to the wait buffer.
	q.waitBufAppend(b)
	_, notifyEcho := q.pushWaitBufLocked(l)
	return notifyEcho
}

// pushWaitBufLocked fills the queue's read buffer with data from the wait
// buffer.
// The returned boolean indicates whether any data was echoed back.
//
// Preconditions:
//   - l.termiosMu must be held for reading.
//   - q.mu must be locked.
func (q *queue) pushWaitBufLocked(l *lineDiscipline) (int, bool) {
	if q.waitBufLen == 0 {
		return 0, false
	}

	// Move data from the wait to the read buffer.
	var total int
	var i int
	var notifyEcho bool
	for i = 0; i < len(q.waitBuf); i++ {
		n, echo := q.transform(l, q, q.waitBuf[i])
		total += n
		notifyEcho = notifyEcho || echo
		if n != len(q.waitBuf[i]) {
			// The read buffer filled up without consuming the
			// entire buffer.
			q.waitBuf[i] = q.waitBuf[i][n:]
			break
		}
	}

	// Update wait buffer based on consumed data.
	q.waitBuf = q.waitBuf[i:]
	q.waitBufLen -= uint64(total)

	return total, notifyEcho
}

// Precondition: q.mu must be locked.
func (q *queue) waitBufAppend(b []byte) {
	q.waitBuf = append(q.waitBuf, b)
	q.waitBufLen += uint64(len(b))
}
