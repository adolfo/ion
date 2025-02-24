package biz

import (
	"sync"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/grpc/ion"
)

// Room represents a Room which manage peers
type Room struct {
	sync.RWMutex
	sid    string
	sfunid string
	peers  map[string]*Peer
}

// newRoom creates a new room instance
func newRoom(sid string, sfunid string) *Room {
	r := &Room{
		sid:    sid,
		sfunid: sfunid,
		peers:  make(map[string]*Peer),
	}
	return r
}

// SID room id
func (r *Room) SID() string {
	return r.sid
}

// addPeer add a peer to room
func (r *Room) addPeer(p *Peer) {

	event := &ion.PeerEvent{
		State: ion.PeerEvent_JOIN,
		Peer: &ion.Peer{
			Sid:  r.sid,
			Uid:  p.uid,
			Info: p.info,
		},
	}
	r.sendPeerEvent(event)

	// Send the peer info in the existing room
	// to the newly added peer.
	for _, peer := range r.getPeers() {
		event := &ion.PeerEvent{
			State: ion.PeerEvent_JOIN,
			Peer: &ion.Peer{
				Sid:  r.sid,
				Uid:  peer.uid,
				Info: peer.info,
			},
		}
		p.sendPeerEvent(event)

		if peer.lastStreamEvent != nil {
			p.sendStreamEvent(peer.lastStreamEvent)
		}
	}

	r.Lock()
	r.peers[p.uid] = p
	r.Unlock()
}

// getPeer get a peer by peer id
func (r *Room) getPeer(uid string) *Peer {
	r.RLock()
	defer r.RUnlock()
	return r.peers[uid]
}

// getPeers get peers in the room
func (r *Room) getPeers() map[string]*Peer {
	r.RLock()
	defer r.RUnlock()
	return r.peers
}

// delPeer delete a peer in the room
func (r *Room) delPeer(uid string) int {
	r.Lock()
	delete(r.peers, uid)
	r.Unlock()

	event := &ion.PeerEvent{
		State: ion.PeerEvent_LEAVE,
		Peer: &ion.Peer{
			Sid: r.sid,
			Uid: uid,
		},
	}
	r.sendPeerEvent(event)

	return len(r.peers)
}

// count return count of peers in room
func (r *Room) count() int {
	r.RLock()
	defer r.RUnlock()
	return len(r.peers)
}

func (r *Room) sendPeerEvent(event *ion.PeerEvent) {
	peers := r.getPeers()
	for _, p := range peers {
		if err := p.sendPeerEvent(event); err != nil {
			log.Errorf("send data to peer(%s) error: %v", p.uid, err)
		}
	}
}

func (r *Room) sendStreamEvent(event *ion.StreamEvent) {
	peers := r.getPeers()
	for _, p := range peers {
		if err := p.sendStreamEvent(event); err != nil {
			log.Errorf("send data to peer(%s) error: %v", p.uid, err)
		}
	}
}

func (r *Room) sendMessage(msg *ion.Message) {
	from := msg.From
	to := msg.To
	data := msg.Data
	log.Debugf("Room.onMessage %v => %v, data: %v", from, to, data)
	peers := r.getPeers()
	for id, p := range peers {
		if id == to || to == "all" || to == r.sid {
			if err := p.sendMessage(msg); err != nil {
				log.Errorf("send msg to peer(%s) error: %v", p.uid, err)
			}
		}
	}
}
