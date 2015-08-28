package hostport_test

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mistifyio/lochness/pkg/hostport"
	logx "github.com/mistifyio/mistify-logrus-ext"
)

func ExampleSplit() {
	examples := []string{
		"localhost",
		"localhost:1234",
		"[localhost]",
		"[localhost]:1234",
		"2001:db8:85a3:8d3:1319:8a2e:370:7348",
		"[2001:db8:85a3:8d3:1319:8a2e:370:7348]",
		"[2001:db8:85a3:8d3:1319:8a2e:370:7348]:443",
		"2001:db8:85a3:8d3:1319:8a2e:370:7348:443",
		":1234",
		"",
		":::",
		"foo:1234:bar",
		"[2001:db8:85a3:8d3:1319:8a2e:370:7348",
		"[localhost",
		"2001:db8:85a3:8d3:1319:8a2e:370:7348]",
		"localhost]",
		"[loca[lhost]:1234",
		"[loca]lhost]:1234",
		"[localhost]:1234]",
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintln(w, "HOSTPORT\tHOST\tPORT\tERR")
	fmt.Fprintln(w, "========\t====\t====\t===")

	for _, hp := range examples {
		host, port, err := hostport.Split(hp)

		fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", hp, host, port, err)
	}
	logx.LogReturnedErr(w.Flush, nil, "failed to flush tabwriter")

	// Output:
	// HOSTPORT					HOST					PORT	ERR
	// ========					====					====	===
	// localhost					localhost					<nil>
	// localhost:1234					localhost				1234	<nil>
	// [localhost]					localhost					<nil>
	// [localhost]:1234				localhost				1234	<nil>
	// 2001:db8:85a3:8d3:1319:8a2e:370:7348		2001:db8:85a3:8d3:1319:8a2e:370		7348	<nil>
	// [2001:db8:85a3:8d3:1319:8a2e:370:7348]		2001:db8:85a3:8d3:1319:8a2e:370:7348		<nil>
	// [2001:db8:85a3:8d3:1319:8a2e:370:7348]:443	2001:db8:85a3:8d3:1319:8a2e:370:7348	443	<nil>
	// 2001:db8:85a3:8d3:1319:8a2e:370:7348:443	2001:db8:85a3:8d3:1319:8a2e:370:7348	443	<nil>
	// :1234											1234	<nil>
	// 												<nil>
	// :::						::						<nil>
	// foo:1234:bar					foo:1234				bar	<nil>
	// [2001:db8:85a3:8d3:1319:8a2e:370:7348								missing ']'
	// [localhost											missing ']'
	// 2001:db8:85a3:8d3:1319:8a2e:370:7348]								missing '['
	// localhost]											missing '['
	// [loca[lhost]:1234										too many '['
	// [loca]lhost]:1234										too many ']'
	// [localhost]:1234]										too many ']'
}
