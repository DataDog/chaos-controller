// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// NetlinkAdapter is an interface being able to read
// the host network interfaces information
type NetlinkAdapter interface {
	LinkList(useLocalhost bool, log *zap.SugaredLogger) ([]NetlinkLink, error)
	LinkByIndex(index int) (NetlinkLink, error)
	LinkByName(name string) (NetlinkLink, error)
	DefaultRoutes() ([]NetlinkRoute, error)
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

// NewNetlinkAdapter returns a standard netlink adapter
func NewNetlinkAdapter() NetlinkAdapter {
	return netlinkAdapter{}
}

// LinkList lists links used in the routing tables for IPv4 only
// the useLocalhost parameter, when set to true, means we will return the localhost interface, if found
func (a netlinkAdapter) LinkList(useLocalhost bool, log *zap.SugaredLogger) ([]NetlinkLink, error) {
	// retrieve links from indexes and cast them
	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("error listing links: %w", err)
	}

	nlinks := []NetlinkLink{}

	for _, link := range links {
		log.Debugw("listing available links...", "linkName", link.Attrs().Name)
		// ignore non local ethernet interfaces according to
		// the v197 systemd/udev naming standards
		// https://systemd.io/PREDICTABLE_INTERFACE_NAMES/
		if !strings.HasPrefix(link.Attrs().Name, "eno") &&
			!strings.HasPrefix(link.Attrs().Name, "ens") &&
			!strings.HasPrefix(link.Attrs().Name, "enp") &&
			!strings.HasPrefix(link.Attrs().Name, "enx") &&
			!strings.HasPrefix(link.Attrs().Name, "eth") &&
			!strings.HasPrefix(link.Attrs().Name, "lo") {
			continue
		}

		if !useLocalhost && strings.HasPrefix(link.Attrs().Name, "lo") {
			continue
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

func (a netlinkAdapter) DefaultRoutes() ([]NetlinkRoute, error) {
	defaultRoutes := []NetlinkRoute{}

	routes, err := a.listRoutes()
	if err != nil {
		return nil, fmt.Errorf("error listing routes: %w", err)
	}

	// find the default route, the one with no source nor destination
	for _, route := range routes {
		if route.Dst == nil && route.Gw != nil {
			link, err := netlink.LinkByIndex(route.LinkIndex)
			if err != nil {
				return nil, fmt.Errorf("error identifying default route link: %w", err)
			}

			defaultRoutes = append(defaultRoutes, netlinkRoute{
				link: newNetlinkLink(link),
				gw:   route.Gw,
			})
		}
	}

	if len(defaultRoutes) == 0 {
		return nil, fmt.Errorf("error getting default route: not found")
	}

	return defaultRoutes, nil
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

func (r netlinkRoute) String() string {
	return fmt.Sprintf("%s on %s", r.gw, r.link.Name())
}
