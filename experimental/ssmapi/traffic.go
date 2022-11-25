package ssmapi

import (
	"net"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/experimental/trackerconn"
	N "github.com/sagernet/sing/common/network"

	"go.uber.org/atomic"
)

type TrafficManager struct {
	nodeTags       map[string]bool
	nodeUsers      map[string]bool
	globalUplink   *atomic.Int64
	globalDownlink *atomic.Int64
	userAccess     sync.Mutex
	userUplink     map[string]*atomic.Int64
	userDownlink   map[string]*atomic.Int64
}

func NewTrafficManager(nodes []Node) *TrafficManager {
	manager := &TrafficManager{
		nodeTags:       make(map[string]bool),
		globalUplink:   atomic.NewInt64(0),
		globalDownlink: atomic.NewInt64(0),
		userUplink:     make(map[string]*atomic.Int64),
		userDownlink:   make(map[string]*atomic.Int64),
	}
	for _, node := range nodes {
		manager.nodeTags[node.Tag()] = true
	}
	return manager
}

func (s *TrafficManager) UpdateUsers(users []string) {
	nodeUsers := make(map[string]bool)
	for _, user := range users {
		nodeUsers[user] = true
	}
	s.nodeUsers = nodeUsers
}

func (s *TrafficManager) userCounter(user string) (*atomic.Int64, *atomic.Int64) {
	s.userAccess.Lock()
	defer s.userAccess.Unlock()
	upCounter, loaded := s.userUplink[user]
	if !loaded {
		upCounter = atomic.NewInt64(0)
		s.userUplink[user] = upCounter
	}
	downCounter, loaded := s.userDownlink[user]
	if !loaded {
		downCounter = atomic.NewInt64(0)
		s.userDownlink[user] = downCounter
	}
	return upCounter, downCounter
}

func (s *TrafficManager) RoutedConnection(metadata adapter.InboundContext, conn net.Conn) net.Conn {
	var readCounter []*atomic.Int64
	var writeCounter []*atomic.Int64
	if s.nodeTags[metadata.Inbound] {
		readCounter = append(readCounter, s.globalUplink)
		writeCounter = append(writeCounter, s.globalDownlink)
	}
	if s.nodeUsers[metadata.User] {
		upCounter, downCounter := s.userCounter(metadata.User)
		readCounter = append(readCounter, upCounter)
		writeCounter = append(writeCounter, downCounter)
	}
	if len(readCounter) > 0 {
		return trackerconn.New(conn, readCounter, writeCounter)
	}
	return conn
}

func (s *TrafficManager) RoutedPacketConnection(metadata adapter.InboundContext, conn N.PacketConn) N.PacketConn {
	var readCounter []*atomic.Int64
	var writeCounter []*atomic.Int64
	if s.nodeTags[metadata.Inbound] {
		readCounter = append(readCounter, s.globalUplink)
		writeCounter = append(writeCounter, s.globalDownlink)
	}
	if s.nodeUsers[metadata.User] {
		upCounter, downCounter := s.userCounter(metadata.User)
		readCounter = append(readCounter, upCounter)
		writeCounter = append(writeCounter, downCounter)
	}
	if len(readCounter) > 0 {
		return trackerconn.NewPacket(conn, readCounter, writeCounter)
	}
	return conn
}

func (s *TrafficManager) ReadUser(user string) (uplink int64, downlink int64) {
	s.userAccess.Lock()
	defer s.userAccess.Unlock()
	upCounter, upLoaded := s.userUplink[user]
	downCounter, downLoaded := s.userDownlink[user]
	if upLoaded {
		uplink = upCounter.Load()
	}
	if downLoaded {
		downlink = downCounter.Load()
	}
	return
}

func (s *TrafficManager) ReadUsers(users []string) (uplinkList []int64, downlinkList []int64) {
	s.userAccess.Lock()
	defer s.userAccess.Unlock()
	for _, user := range users {
		var uplink, downlink int64
		upCounter, upLoaded := s.userUplink[user]
		downCounter, downLoaded := s.userDownlink[user]
		if upLoaded {
			uplink = upCounter.Load()
		}
		if downLoaded {
			downlink = downCounter.Load()
		}
		uplinkList = append(uplinkList, uplink)
		downlinkList = append(downlinkList, downlink)
	}
	return
}

func (s *TrafficManager) ReadGlobal() (uplink int64, downlink int64) {
	return s.globalUplink.Load(), s.globalDownlink.Load()
}
