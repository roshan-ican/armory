package live

import "sync"

// Hub tells every subscriber that something changed; subscribers re-fetch what they show.
type AdminAlert struct {
	Text   string
	Urgent bool
}

type Hub struct {
	mu     sync.Mutex
	subs   map[chan struct{}]struct{}
	speech map[chan string]struct{}
	alerts map[chan AdminAlert]struct{}
}

func NewHub() *Hub {
	return &Hub{
		subs:   make(map[chan struct{}]struct{}),
		speech: make(map[chan string]struct{}),
		alerts: make(map[chan AdminAlert]struct{}),
	}
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

func (h *Hub) SubscribeSpeech() (<-chan string, func()) {
	ch := make(chan string, 4)
	h.mu.Lock()
	h.speech[ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.speech, ch)
		h.mu.Unlock()
	}
}

func (h *Hub) Say(text string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.speech {
		select {
		case ch <- text:
		default:
		}
	}
}

func (h *Hub) SubscribeAlerts() (<-chan AdminAlert, func()) {
	ch := make(chan AdminAlert, 4)
	h.mu.Lock()
	h.alerts[ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.alerts, ch)
		h.mu.Unlock()
	}
}

func (h *Hub) Alert(text string) {
	h.tell(AdminAlert{Text: text, Urgent: true})
}

func (h *Hub) Notice(text string) {
	h.tell(AdminAlert{Text: text})
}

func (h *Hub) tell(alert AdminAlert) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.alerts {
		select {
		case ch <- alert:
		default:
		}
	}
}
