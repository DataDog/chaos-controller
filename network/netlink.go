// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package network

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

// NetlinkAdapter is an interface being able to read
// the host network interfaces information
type NetlinkAdapter interface {
	LinkList() ([]NetlinkLink, error)
	LinkByIndex(index int) (NetlinkLink, error)
	LinkByName(name string) (NetlinkLink, error)
	RoutesForIP(ip *net.IPNet) ([]NetlinkRoute, error)
	DefaultRoute() (NetlinkRoute, error)
}

type netlinkAdapter struct{}

func (a netlinkAdapter) listRoutes() ([]netlink.Route, error) {
	// get the handler
	handler, err := netlink.NewHandle()
	if err != nil {
		return nil, err
	}

	// list routes for all interfaces using IPv4
	// cf. https://godoc.org/golang.org/x/sys/unix#AF_INET for value 2
	routes, err := handler.RouteList(nil, 2)
	if err != nil {
		return nil, err
	}

	return routes, nil
}

// NewNetlinkAdapter returns a standard netlink adapter
func NewNetlinkAdapter() NetlinkAdapter {
	return netlinkAdapter{}
}

// LinkList lists links used in the routing table for IPv4 only
func (a netlinkAdapter) LinkList() ([]NetlinkLink, error) {
	// get routes
	routes, err := a.listRoutes()
	if err != nil {
		return nil, fmt.Errorf("error listing routes: %w", err)
	}

	// store links indexes
	linksIndexes := map[int]struct{}{}

	for _, route := range routes {
		if _, found := linksIndexes[route.LinkIndex]; !found {
			// ignore any "invalid" link index
			// the link index can be 0 for blackhole routes for instance (eq. "*" interface in the routing table)
			if route.LinkIndex > 0 {
				linksIndexes[route.LinkIndex] = struct{}{}
			}
		}
	}

	// retrieve links from indexes and cast them
	nlinks := []NetlinkLink{}

	for linkIndex := range linksIndexes {
		link, err := netlink.LinkByIndex(linkIndex)
		if err != nil {
			return nil, fmt.Errorf("error getting link with index %d: %w", linkIndex, err)
		}

		nlinks = append(nlinks, newNetlinkLink(link))
	}

	return nlinks, nil
}

func (a netlinkAdapter) LinkByIndex(index int) (NetlinkLink, error) {
	link, err := netlink.LinkByIndex(index)
	if err != nil {
		return nil, err
	}

	return newNetlinkLink(link), nil
}

func (a netlinkAdapter) LinkByName(name string) (NetlinkLink, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}

	return newNetlinkLink(link), nil
}

func (a netlinkAdapter) RoutesForIP(ip *net.IPNet) ([]NetlinkRoute, error) {
	r := []NetlinkRoute{}

	// get the handler
	handler, err := netlink.NewHandle()
	if err != nil {
		return nil, err
	}

	// get routes for given ip
	routes, err := handler.RouteGet(ip.IP)
	if err != nil {
		return nil, err
	}

	// convert netlink routes to interfaces
	for _, route := range routes {
		link, err := netlink.LinkByIndex(route.LinkIndex)
		if err != nil {
			return nil, err
		}

		r = append(r, netlinkRoute{
			link: newNetlinkLink(link),
			gw:   route.Gw,
		})
	}

	return r, nil
}

func (a netlinkAdapter) DefaultRoute() (NetlinkRoute, error) {
	routes, err := a.listRoutes()
	if err != nil {
		return nil, fmt.Errorf("error listing routes: %w", err)
	}

	// find the default route, the one with no source nor destination
	for _, route := range routes {
		if route.Dst == nil {
			link, err := netlink.LinkByIndex(route.LinkIndex)
			if err != nil {
				return nil, fmt.Errorf("error identifying default route link: %w", err)
			}

			return netlinkRoute{
				link: newNetlinkLink(link),
				gw:   route.Gw,
			}, nil
		}
	}

	return nil, fmt.Errorf("error getting default route: not found")
}

// NetlinkLink is a host interface
type NetlinkLink interface {
	Name() string
	SetTxQLen(qlen int) error
	TxQLen() int
}

type netlinkLink struct {
	name   string
	txQLen int
}

func (l netlinkLink) Name() string {
	return l.name
}

func (l *netlinkLink) SetTxQLen(qlen int) error {
	link, err := netlink.LinkByName(l.name)
	if err != nil {
		return err
	}

	l.txQLen = qlen

	return netlink.LinkSetTxQLen(link, qlen)
}

func (l netlinkLink) TxQLen() int {
	return l.txQLen
}

func newNetlinkLink(link netlink.Link) *netlinkLink {
	return &netlinkLink{
		name:   link.Attrs().Name,
		txQLen: link.Attrs().TxQLen,
	}
}

// NetlinkRoute is a route attached to a host interface
type NetlinkRoute interface {
	Link() NetlinkLink
	Gateway() net.IP
}

type netlinkRoute struct {
	link NetlinkLink
	gw   net.IP
}

func (r netlinkRoute) Link() NetlinkLink {
	return r.link
}

func (r netlinkRoute) Gateway() net.IP {
	return r.gw
}
