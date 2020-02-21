// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package network

import (
	"net"

	"github.com/vishvananda/netlink"
)

type NetlinkAdapter interface {
	LinkList() ([]NetlinkLink, error)
	LinkByIndex(index int) (NetlinkLink, error)
	LinkByName(name string) (NetlinkLink, error)
	RoutesForIP(ip *net.IPNet) ([]NetlinkRoute, error)
}

type netlinkAdapter struct{}

func NewNetlinkAdapter() NetlinkAdapter {
	return netlinkAdapter{}
}

func (a netlinkAdapter) LinkList() ([]NetlinkLink, error) {
	// list links
	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	// cast to interface
	nlinks := []NetlinkLink{}
	for _, link := range links {
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
		})
	}

	return r, nil
}

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

type NetlinkRoute interface {
	Link() NetlinkLink
}

type netlinkRoute struct {
	link NetlinkLink
}

func (r netlinkRoute) Link() NetlinkLink {
	return r.link
}
