package main

import (
	"fmt"
	"io"
)

//line nftables.ego:1
func nftWrite(w io.Writer, ip string, groups groupMap, guests guestMap) error {
//line nftables.ego:2
	_, _ = fmt.Fprintf(w, "\nflush ruleset\n\ntable ip filter {\n  ")
//line nftables.ego:5
	for id, fwg := range groups {
//line nftables.ego:6
		_, _ = fmt.Fprintf(w, "\n  # FWGroupID=")
//line nftables.ego:6
		_, _ = fmt.Fprintf(w, "%v", id)
//line nftables.ego:7
		_, _ = fmt.Fprintf(w, "\n  chain g")
//line nftables.ego:7
		_, _ = fmt.Fprintf(w, "%v", fwg.num)
//line nftables.ego:7
		_, _ = fmt.Fprintf(w, " {")
//line nftables.ego:7
		for _, rule := range fwg.rules {
//line nftables.ego:8
			_, _ = fmt.Fprintf(w, "\n      ")
//line nftables.ego:8
			_, _ = fmt.Fprintf(w, "%v", rule)
//line nftables.ego:8
			_, _ = fmt.Fprintf(w, " accept ")
//line nftables.ego:8
		}
//line nftables.ego:9
		_, _ = fmt.Fprintf(w, "\n  }\n  set s")
//line nftables.ego:10
		_, _ = fmt.Fprintf(w, "%v", fwg.num)
//line nftables.ego:10
		_, _ = fmt.Fprintf(w, " {\n    type ipv4_addr")
//line nftables.ego:11
		if len(fwg.ips) > 0 {
//line nftables.ego:12
			_, _ = fmt.Fprintf(w, "\n    elements = { ")
//line nftables.ego:12
			for _, ip := range fwg.ips {
//line nftables.ego:13
				_, _ = fmt.Fprintf(w, "\n      ")
//line nftables.ego:13
				_, _ = fmt.Fprintf(w, "%v", ip)
//line nftables.ego:13
				_, _ = fmt.Fprintf(w, ", ")
//line nftables.ego:13
			}
//line nftables.ego:14
			_, _ = fmt.Fprintf(w, "\n    }")
//line nftables.ego:14
		}
//line nftables.ego:15
		_, _ = fmt.Fprintf(w, "\n  }\n  ")
//line nftables.ego:16
	}
//line nftables.ego:17
	_, _ = fmt.Fprintf(w, "\n  chain input {\n    type filter hook input priority 0;\n\n    # allow established/related connections\n    ct state {established, related} accept\n\n    # early drop of invalid connections\n    ct state invalid drop\n\n    # allow from loopback\n    iifname lo accept\n\n    # allow icmp\n    ip protocol icmp accept\n\n    # allow lochness hv traffic\n    ip daddr ")
//line nftables.ego:33
	_, _ = fmt.Fprintf(w, "%v", ip)
//line nftables.ego:33
	_, _ = fmt.Fprintf(w, " accept\n\n  }\n\n  chain forward {\n    type filter hook forward priority 0;\n    drop\n  }\n\n  chain output {\n    type filter hook output priority 0;\n  }\n}\n\n")
//line nftables.ego:47
	if len(guests) > 0 {
//line nftables.ego:48
		_, _ = fmt.Fprintf(w, "\n# Allow traffic to guests as specified by FWGroups\nadd rule filter input ip daddr vmap { ")
//line nftables.ego:49
		for ip, fwgIndex := range guests {
//line nftables.ego:50
			_, _ = fmt.Fprintf(w, "\n    ")
//line nftables.ego:50
			_, _ = fmt.Fprintf(w, "%v", ip)
//line nftables.ego:50
			_, _ = fmt.Fprintf(w, " : jump g")
//line nftables.ego:50
			_, _ = fmt.Fprintf(w, "%v", fwgIndex)
//line nftables.ego:50
			_, _ = fmt.Fprintf(w, ", ")
//line nftables.ego:50
		}
//line nftables.ego:51
		_, _ = fmt.Fprintf(w, "\n}\n")
//line nftables.ego:52
	}
//line nftables.ego:53
	_, _ = fmt.Fprintf(w, "\n\n# everything else\nadd rule filter input reject with icmp type port-unreachable\n")
	return nil
}
