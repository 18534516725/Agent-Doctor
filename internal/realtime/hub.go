package realtime

import "sync"

type Event struct {
	ID        uint64 `json:"id"`
	Kind      string `json:"kind"`
	SessionID string `json:"sessionId,omitempty"`
}

type Hub struct {
	mu          sync.Mutex
	nextID      uint64
	capacity    int
	history     []Event
	nextSubID   uint64
	subscribers map[uint64]chan Event
}

func NewHub(capacity int) *Hub {
	if capacity < 1 {
		capacity = 1
	}
	return &Hub{capacity: capacity, subscribers: map[uint64]chan Event{}}
}

func (hub *Hub) Publish(event Event) Event {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	hub.nextID++
	event.ID = hub.nextID
	hub.history = append(hub.history, event)
	if len(hub.history) > hub.capacity {
		hub.history = append([]Event(nil), hub.history[len(hub.history)-hub.capacity:]...)
	}
	for id, channel := range hub.subscribers {
		select {
		case channel <- event:
		default:
			close(channel)
			delete(hub.subscribers, id)
		}
	}
	return event
}

func (hub *Hub) Subscribe(afterID uint64, buffer int) (<-chan Event, func()) {
	if buffer < 1 {
		buffer = 1
	}
	hub.mu.Lock()
	hub.nextSubID++
	id := hub.nextSubID
	replayCount := 0
	for _, event := range hub.history {
		if event.ID > afterID {
			replayCount++
		}
	}
	if buffer < replayCount {
		buffer = replayCount
	}
	channel := make(chan Event, buffer)
	for _, event := range hub.history {
		if event.ID > afterID {
			channel <- event
		}
	}
	hub.subscribers[id] = channel
	hub.mu.Unlock()
	var once sync.Once
	return channel, func() {
		once.Do(func() {
			hub.mu.Lock()
			if existing, ok := hub.subscribers[id]; ok {
				delete(hub.subscribers, id)
				close(existing)
			}
			hub.mu.Unlock()
		})
	}
}
