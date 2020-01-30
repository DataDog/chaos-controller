package injector

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/container"
	"github.com/DataDog/chaos-fi-controller/helpers"
	"github.com/vishvananda/netlink"
)

const tcPath = "/sbin/tc"

// executeTcCommand executes the given args using the tc command
// and returns a wrapped error containing both the error returned by the execution and
// the stderr content
func (i NetworkLatencyInjector) executeTcCommand(args string) error {
	i.Log.Infof("running tc command: %s %s", tcPath, args)

	// parse args and execute
	split := strings.Split(args, " ")
	stderr := &bytes.Buffer{}
	cmd := exec.Command(tcPath, split...)
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("%w: %s", err, stderr.String())
	}

	return err
}

// NetworkLatencyInjector describes a network latency
type NetworkLatencyInjector struct {
	ContainerInjector
	Spec *v1beta1.NetworkLatencySpec
}

func (i NetworkLatencyInjector) getInterfacesByIP() (map[string][]*net.IPNet, error) {
	linkByIP := map[string][]*net.IPNet{}

	if len(i.Spec.Hosts) > 0 {
		i.Log.Info("auto-detecting interfaces to apply latency to...")
		// resolve hosts
		ips, err := helpers.ResolveHost(i.Spec.Hosts)
		if err != nil {
			return nil, fmt.Errorf("can't resolve given hosts: %w", err)
		}

		// create the netlink handler
		handler, err := netlink.NewHandle()
		if err != nil {
			return nil, fmt.Errorf("can't get the netlink handler: %w", err)
		}

		// get the association between IP and interfaces to know
		// which interfaces we have to inject latency to
		for _, ip := range ips {
			// get routes for resolved destination IP
			routes, err := handler.RouteGet(ip.IP)
			if err != nil {
				return nil, fmt.Errorf("can't get route for IP %s: %w", ip.String(), err)
			}

			// for each route, get the related interface and add it to the association
			// between interfaces and IPs
			for _, route := range routes {
				// retrieve link from route
				link, err := netlink.LinkByIndex(route.LinkIndex)
				if err != nil {
					return nil, fmt.Errorf("can't get link for route %s and IP %s: %w", route.String(), ip.String(), err)
				}

				// store association, initialize the map entry if not present yet
				i.Log.Infof("IP %s belongs to interface %s", ip.String(), link.Attrs().Name)
				if _, ok := linkByIP[link.Attrs().Name]; !ok {
					linkByIP[link.Attrs().Name] = []*net.IPNet{}
				}
				linkByIP[link.Attrs().Name] = append(linkByIP[link.Attrs().Name], ip)
			}
		}
	} else {
		i.Log.Info("no hosts specified, all interfaces will be impacted")

		// prepare links/IP association by pre-creating links
		links, err := netlink.LinkList()
		if err != nil {
			i.Log.Fatalf("can't list links: %w", err)
		}
		for _, link := range links {
			i.Log.Info("adding interface %s", link.Attrs().Name)
			linkByIP[link.Attrs().Name] = []*net.IPNet{}
		}
	}

	return linkByIP, nil
}

// Inject injects network latency according to the current spec
func (i NetworkLatencyInjector) Inject() {
	i.Log.Info("injecting latency")

	delay := time.Duration(i.Spec.Delay) * time.Millisecond
	parent := "root"

	// enter container network namespace
	// and defer the exit on return
	c, err := container.New(i.ContainerID)
	if err != nil {
		i.Log.Fatalw("unable to load the given container", "error", err, "id", i.ContainerID)
	}
	err = c.EnterNetworkNamespace()
	if err != nil {
		i.Log.Fatalw("unable to enter the given container network namespace", "error", err, "id", i.ContainerID)
	}
	defer func() {
		err := container.ExitNetworkNamespace()
		if err != nil {
			i.Log.Fatalw("unable to exit the given container network namespace", "error", err, "id", i.ContainerID)
		}
	}()

	i.Log.Info("auto-detecting interfaces to apply latency to...")
	linkByIP, err := i.getInterfacesByIP()
	if err != nil {
		i.Log.Fatalw("can't get interfaces per IP listing: %w", err)
	}

	// for each link/ip association, add latency
	for linkName, ips := range linkByIP {
		// retrieve link from name
		link, err := netlink.LinkByName(linkName)
		if err != nil {
			i.Log.Fatalf("can't retrieve link %s: %w", linkName, err)
		}

		// if at least one IP has been specified, we need to create a prio qdisc to be able to apply
		// a filter and a delay only on traffic going to those IP
		clearTxQlen := false
		if len(ips) > 0 {
			// set the tx qlen if not already set as it is required to create a prio qdisc without dropping
			// all the outgoing traffic
			// this qlen will be removed once the injection is done if it was not present before
			if link.Attrs().TxQLen == 0 {
				i.Log.Infof("setting tx qlen for interface %s", link.Attrs().Name)
				clearTxQlen = true
				if err := netlink.LinkSetTxQLen(link, 1000); err != nil {
					i.Log.Fatalf("can't set tx queue length on interface %s: %w", link.Attrs().Name, err)
				}
			}

			// create a new qdisc for the given interface of type prio with 4 bands instead of 3
			// we keep the default priomap, the extra band will be used to filter traffic going to the specified IP
			// we only create this qdisc if we want to target traffic going to some hosts only, it avoids to add delay to
			// all the outgoing traffic
			parent = "parent 1:4"
			if err := i.executeTcCommand(fmt.Sprintf("qdisc add dev %s root handle 1: prio bands 4 priomap 1 2 2 2 1 2 0 0 1 1 1 1 1 1 1 1", link.Attrs().Name)); err != nil {
				i.Log.Fatalf("can't create a new qdisc for interface %s: %w", link.Attrs().Name, err)
			}
		}

		// add delay
		if err := i.executeTcCommand(fmt.Sprintf("qdisc add dev %s %s netem delay %s", link.Attrs().Name, parent, delay.String())); err != nil {
			i.Log.Fatalf("can't add delay to the newly created qdisc for interface %s: %w", link.Attrs().Name, err)
		}

		// if only some hosts are targeted, redirect the traffic to the extra band created earlier
		// where the delay is applied
		if len(ips) > 0 {
			for _, ip := range ips {
				if err := i.executeTcCommand(fmt.Sprintf("filter add dev %s parent 1:0 protocol ip prio 1 u32 match ip dst %s flowid 1:4", link.Attrs().Name, ip.String())); err != nil {
					i.Log.Fatalf("can't add a filter to interface %s: %w", link.Attrs().Name, err)
				}
			}
		}

		// reset the interface transmission queue length once filters have been created
		if clearTxQlen {
			i.Log.Infof("clearing tx qlen for interface %s", link.Attrs().Name)
			if err := netlink.LinkSetTxQLen(link, 0); err != nil {
				i.Log.Fatalf("can't clear %s link transmission queue length: %w", link.Attrs().Name, err)
			}
		}
	}
}

// Clean cleans the injected latency
func (i NetworkLatencyInjector) Clean() {
	i.Log.Info("cleaning latency")

	// enter container network namespace
	// and defer the exit on return
	c, err := container.New(i.ContainerID)
	if err != nil {
		i.Log.Fatalw("unable to load the given container", "error", err, "id", i.ContainerID)
	}
	err = c.EnterNetworkNamespace()
	if err != nil {
		i.Log.Fatalw("unable to enter the given container network namespace", "error", err, "id", i.ContainerID)
	}
	defer func() {
		err := container.ExitNetworkNamespace()
		if err != nil {
			i.Log.Fatalw("unable to exit the given container network namespace", "error", err, "id", i.ContainerID)
		}
	}()

	linkByIP, err := i.getInterfacesByIP()
	if err != nil {
		i.Log.Fatalf("can't get interfaces per IP map: %w", err)
	}

	for linkName := range linkByIP {
		// retrieve link from name
		link, err := netlink.LinkByName(linkName)
		if err != nil {
			i.Log.Fatalf("can't retrieve link %s: %w", linkName, err)
		}

		i.Log.Infof("clearing root qdisc for interface %s", link.Attrs().Name)
		if err := i.executeTcCommand(fmt.Sprintf("qdisc del dev %s root", link.Attrs().Name)); err != nil {
			i.Log.Fatalf("can't delete the %s link qdisc: %w", link.Attrs().Name, err)
		}
	}
}
