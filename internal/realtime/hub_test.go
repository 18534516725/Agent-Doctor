package realtime

import "testing"

func TestHubPublishesMonotonicEventsAndReplaysAfterID(t *testing.T) {
	hub := NewHub(3)
	first := hub.Publish(Event{Kind: "conversation.saved", SessionID: "s1"})
	second := hub.Publish(Event{Kind: "connection.changed"})
	third := hub.Publish(Event{Kind: "conversation.saved", SessionID: "s2"})
	if first.ID != 1 || second.ID != 2 || third.ID != 3 {
		t.Fatalf("ids=%d,%d,%d", first.ID, second.ID, third.ID)
	}
	ch, cancel := hub.Subscribe(1, 4)
	defer cancel()
	for _, want := range []uint64{2, 3} {
		got := <-ch
		if got.ID != want {
			t.Fatalf("replay id=%d want=%d", got.ID, want)
		}
	}
	fourth := hub.Publish(Event{Kind: "analysis.changed"})
	if got := <-ch; got.ID != fourth.ID {
		t.Fatalf("live id=%d want=%d", got.ID, fourth.ID)
	}
}

func TestHubEvictsSlowSubscribersInsteadOfBlockingPublish(t *testing.T) {
	hub := NewHub(2)
	channel, _ := hub.Subscribe(0, 1)
	hub.Publish(Event{Kind: "one"})
	hub.Publish(Event{Kind: "two"})
	if _, ok := <-channel; !ok {
		return
	}
	if _, ok := <-channel; ok {
		t.Fatal("slow subscriber should be closed")
	}
}
