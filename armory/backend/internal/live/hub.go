package live

import "sync"

// Hub tells every subscriber that something changed; subscribers re-fetch what they show.
type Hub struct {
	mu   sync.Mutex
	subs map[chan struct{}]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[chan struct{}]struct{})}
}

func (h *Hub) Subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
}

func (h *Hub) Publish() {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
