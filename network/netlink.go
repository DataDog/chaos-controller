// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package network

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// NetlinkAdapter is an interface being able to read
// the host network interfaces information
type NetlinkAdapter interface {
	LinkList() ([]NetlinkLink, error)
	LinkByIndex(index int) (NetlinkLink, error)
	LinkByName(name string) (NetlinkLink, error)
	DefaultRoute() (NetlinkRoute, error)
}

type netlinkAdapter struct{}

func (a netlinkAdapter) listRoutes() ([]netlink.Route, error) {
	allRoutes := []netlink.Route{}

	// get the netlink handler
	handler, err := netlink.NewHandle()
	if err != nil {
		return nil, err
	}

	// list routing rules for IPv4
	rules, err := handler.RuleList(unix.AF_INET)
	if err != nil {
		return nil, err
	}

	// get routing tables identifiers from rules so we
	// are able to list all the existing routing tables
	tables := map[int]struct{}{}

	for _, rule := range rules {
		if _, found := tables[rule.Table]; !found {
			tables[rule.Table] = struct{}{}
		}
	}

	// get all the existing routing tables routes
	for table := range tables {
		// NOTE: we are using a magic number here (1024, which comes from the netlink library constants) for MacOS build compatibility
		// netlink.RT_FILTER_TABLE == 1024
		// https://github.com/vishvananda/netlink/blob/v1.1.0/route_linux.go#L34
		routes, err := handler.RouteListFiltered(unix.AF_INET, &netlink.Route{Table: table}, 1024)
		if err != nil {
			return nil, err
		}

		allRoutes = append(allRoutes, routes...)
	}

	return allRoutes, nil
}

func (a netlinkAdapter) getBridgeLinks(bridge netlink.Link) ([]netlink.Link, error) {
	bridgeLinks := []netlink.Link{}

	// get the netlink handler
	handler, err := netlink.NewHandle()
	if err != nil {
		return nil, err
	}

	// list all links links
	links, err := handler.LinkList()
	if err != nil {
		return nil, err
	}

	// get links having the given bridge link as master
	for _, link := range links {
		if link.Attrs().MasterIndex == bridge.Attrs().Index {
			bridgeLinks = append(bridgeLinks, link)
		}
	}

	return bridgeLinks, nil
}

// NewNetlinkAdapter returns a standard netlink adapter
func NewNetlinkAdapter() NetlinkAdapter {
	return netlinkAdapter{}
}

// LinkList lists links used in the routing tables for IPv4 only
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
		// get link from the link index found in the route
		link, err := netlink.LinkByIndex(linkIndex)
		if err != nil {
			return nil, fmt.Errorf("error getting link with index %d: %w", linkIndex, err)
		}
		nlinks = append(nlinks, newNetlinkLink(link))

		// for any bridge link, get bridge slaves (interfaces for which the bridge link is considered as master)
		if link.Type() == "bridge" {
			bridgeLinks, err := a.getBridgeLinks(link)
			if err != nil {
				return nil, fmt.Errorf("error getting bridge links for interface %s: %w", link.Attrs().Name, err)
			}

			for _, bridgeLink := range bridgeLinks {
				nlinks = append(nlinks, newNetlinkLink(bridgeLink))
			}
		}
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
