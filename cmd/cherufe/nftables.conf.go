package main

const ruleset = `flush ruleset

table inet filter {
  {{with $.Sources}}{{range .}}# FWGroupID={{.ID}}
  set g{{.Name}} {
    type ipv4_addr{{if .IPs}}
    elements = { {{range .IPs}}{{.}},{{end}} }{{end}}
  }

  {{end}}{{end}}chain input {
    type filter hook input priority 0;

    # allow established/related connections
    ct state {established, related} accept

    # early drop of invalid connections
    ct state invalid drop

    # allow from loopback
    iifname lo accept

    # allow icmp
    ip protocol icmp accept
    ip6 nexthdr icmpv6 accept

    # allow lochness hv traffic
    # ssh, http, beanstalk, etcd
    ip daddr {{ $.IP }} tcp dport {22, 80, 11300, 7001} accept

    {{with $.Rules}}# allow traffic to guests as specified by FWGroup{{range .}}
    {{.}} accept{{end}}

    {{end}}# everything else
    reject with icmp type port-unreachable
  }

  chain forward {
    type filter hook forward priority 0;
    drop
  }

  chain output {
    type filter hook output priority 0;
  }
}
`
