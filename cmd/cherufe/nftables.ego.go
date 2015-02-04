package main

import (
	"fmt"
	"io"
)

//line nftables.ego:1
func nftWrite(w io.Writer, ip string, sources []group, rules []string) error {
//line nftables.ego:2
	_, _ = fmt.Fprintf(w, "\nflush ruleset\n\ntable inet filter {\n  ")
//line nftables.ego:5
	for _, group := range sources {
//line nftables.ego:6
		_, _ = fmt.Fprintf(w, "\n  # FWGroupID=")
//line nftables.ego:6
		_, _ = fmt.Fprintf(w, "%v", group.ID)
//line nftables.ego:7
		_, _ = fmt.Fprintf(w, "\n  set g")
//line nftables.ego:7
		_, _ = fmt.Fprintf(w, "%v", group.Name)
//line nftables.ego:7
		_, _ = fmt.Fprintf(w, " {\n    type ipv4_addr")
//line nftables.ego:8
		if len(group.IPs) > 0 {
//line nftables.ego:9
			_, _ = fmt.Fprintf(w, "\n    elements = { ")
//line nftables.ego:9
			for i := range group.IPs {
//line nftables.ego:10
				_, _ = fmt.Fprintf(w, "\n      ")
//line nftables.ego:10
				_, _ = fmt.Fprintf(w, "%v", group.IPs[i])
//line nftables.ego:10
				_, _ = fmt.Fprintf(w, ", ")
//line nftables.ego:10
			}
//line nftables.ego:11
			_, _ = fmt.Fprintf(w, "\n    }")
//line nftables.ego:11
		}
//line nftables.ego:12
		_, _ = fmt.Fprintf(w, "\n  }\n  ")
//line nftables.ego:13
	}
//line nftables.ego:14
	_, _ = fmt.Fprintf(w, "\n  chain input {\n    type filter hook input priority 0;\n\n    # allow established/related connections\n    ct state {established, related} accept\n\n    # early drop of invalid connections\n    ct state invalid drop\n\n    # allow from loopback\n    iifname lo accept\n\n    # allow icmp\n    ip protocol icmp accept\n    ip6 nexthdr icmpv6 accept\n\n    # allow lochness hv traffic\n    # ssh, http, beanstalk, etcd\n    ip daddr ")
//line nftables.ego:32
	_, _ = fmt.Fprintf(w, "%v", ip)
//line nftables.ego:32
	_, _ = fmt.Fprintf(w, " accept\n\n    ")
//line nftables.ego:34
	if len(rules) > 0 {
//line nftables.ego:35
		_, _ = fmt.Fprintf(w, "\n    # Allow traffic to guests as specified by FWGroups\n    ")
//line nftables.ego:36
		for i := range rules {
//line nftables.ego:37
			_, _ = fmt.Fprintf(w, "\n    ")
//line nftables.ego:37
			_, _ = fmt.Fprintf(w, "%v", rules[i])
//line nftables.ego:37
			_, _ = fmt.Fprintf(w, " accept")
//line nftables.ego:37
		}
//line nftables.ego:38
		_, _ = fmt.Fprintf(w, "\n    ")
//line nftables.ego:38
	}
//line nftables.ego:39
	_, _ = fmt.Fprintf(w, "\n    # everything else\n    reject with icmp type port-unreachable\n  }\n\n  chain forward {\n    type filter hook forward priority 0;\n    drop\n  }\n\n  chain output {\n    type filter hook output priority 0;\n  }\n}\n")
	return nil
}
