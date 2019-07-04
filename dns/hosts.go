package dns

import (
	"net"
	"strings"
)

type hosts struct {
	mapping []element
}

type element struct {
	host     string
	IP       net.IP
	Resolver []resolver
}

// Get element in Cache, and drop when it expired
func (c *hosts) Get(key interface{}) (net.IP, []resolver) {
	domain := key.(string)

	var host string
	for _, elm := range c.mapping {
		if strings.HasPrefix(elm.host, ".") {
			host = elm.host
			if strings.HasSuffix(domain, host) {
				return elm.IP, elm.Resolver
			}
			host = strings.TrimSuffix(host, ".")
		} else {
			host = elm.host
		}

		if domain == host {
			return elm.IP, elm.Resolver
		}
	}
	return nil, nil
}

type HostMapping struct {
	Host string
	Net  string
	Addr interface{}
}

func NewHosts(hostmappings []HostMapping) *hosts {
	c := &hosts{}

	for _, item := range hostmappings {
		var ip net.IP
		var servers []NameServer

		if item.Net == "mapping" {
			ip = item.Addr.(net.IP)
			servers = nil
		} else {
			ip = nil
			servers = item.Addr.([]NameServer)
		}

		c.mapping = append(c.mapping, element{
			item.Host,
			ip,
			transform(servers),
		})
	}
	return c
}
