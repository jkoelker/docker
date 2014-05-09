package network

import (
	"fmt"
	"github.com/dotcloud/docker/pkg/libcontainer"
	"github.com/dotcloud/docker/pkg/libcontainer/utils"
)

// Ovs is a network strategy that creates an OVS Internal port on an OVS
// bridge and sets it in the container's namespace
type Ovs struct {
}

func (v *Ovs) Create(n *libcontainer.Network, nspid int, context libcontainer.Context) error {
	var (
		bridge string
		prefix string
		exists bool
	)
	if bridge, exists = n.Context["bridge"]; !exists {
		return fmt.Errorf("bridge does not exist in network context")
	}
	if prefix, exists = n.Context["prefix"]; !exists {
		return fmt.Errorf("port prefix does not exist in network context")
	}
	name, err := createPort(prefix, bridge)
	if err != nil {
		return err
	}
	context["ovs-port"] = name
	if err := SetInterfaceInNamespacePid(name, nspid); err != nil {
		return err
	}
	return nil
}

func (v *Ovs) Initialize(config *libcontainer.Network, context libcontainer.Context) error {
	var (
		ovsPort string
		exists  bool
	)
	if ovsPort, exists = context["ovs-port"]; !exists {
		return fmt.Errorf("ovsPort does not exist in network context")
	}
	if err := InterfaceDown(ovsPort); err != nil {
		return fmt.Errorf("interface down %s %s", ovsPort, err)
	}
	if err := ChangeInterfaceName(ovsPort, "eth0"); err != nil {
		return fmt.Errorf("change %s to eth0 %s", ovsPort, err)
	}
	if err := SetInterfaceIp("eth0", config.Address); err != nil {
		return fmt.Errorf("set eth0 ip %s", err)
	}
	if err := SetMtu("eth0", config.Mtu); err != nil {
		return fmt.Errorf("set eth0 mtu to %d %s", config.Mtu, err)
	}
	if err := InterfaceUp("eth0"); err != nil {
		return fmt.Errorf("eth0 up %s", err)
	}
	if config.Gateway != "" {
		if err := SetDefaultGateway(config.Gateway); err != nil {
			return fmt.Errorf("set gateway to %s %s", config.Gateway, err)
		}
	}
	return nil
}

func createPort(prefix string, bridge string) (name string, err error) {
	name, err = utils.GenerateRandomName(prefix, 4)
	if err != nil {
		return
	}
	if err = CreateInternalOVSPort(name, bridge); err != nil {
		return
	}
	return
}
