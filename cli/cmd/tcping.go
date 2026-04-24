// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.
package cmd

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	tcpingCount    int
	tcpingInterval time.Duration
	tcpingTimeout  time.Duration
)

var tcpingCmd = &cobra.Command{
	Use:   "tcping HOST PORT",
	Short: "Ping a TCP port on a remote host",
	Long: `Probe a TCP port by performing a TCP handshake and measuring round-trip latency.

This is useful for verifying TCP connectivity and measuring connection setup time
without needing additional tools like curl in a loop or nmap.

The command runs continuously until interrupted (Ctrl+C) or the specified count
is reached, then prints summary statistics.`,
	Example: `  # Continuously ping a web server on port 443
  kubectl retina tcping example.com 443

  # Send exactly 10 probes with a 500ms interval
  kubectl retina tcping example.com 80 --count 10 --interval 500ms

  # Use a 5-second connection timeout
  kubectl retina tcping 10.0.0.1 8080 --timeout 5s`,
	Args: cobra.ExactArgs(2),
	RunE: runTCPing,
}

func init() {
	Retina.AddCommand(tcpingCmd)
	tcpingCmd.Flags().IntVarP(&tcpingCount, "count", "c", 0, "Number of probes to send (0 = unlimited)")
	tcpingCmd.Flags().DurationVarP(&tcpingInterval, "interval", "i", 1*time.Second, "Interval between probes")
	tcpingCmd.Flags().DurationVarP(&tcpingTimeout, "timeout", "t", 2*time.Second, "TCP connection timeout per probe")
}

type tcpingStats struct {
	sent      int
	succeeded int
	minRTT    time.Duration
	maxRTT    time.Duration
	totalRTT  time.Duration
}

func (s *tcpingStats) record(rtt time.Duration, ok bool) {
	s.sent++
	if !ok {
		return
	}
	s.succeeded++
	s.totalRTT += rtt
	if s.succeeded == 1 || rtt < s.minRTT {
		s.minRTT = rtt
	}
	if rtt > s.maxRTT {
		s.maxRTT = rtt
	}
}

func (s *tcpingStats) avgRTT() time.Duration {
	if s.succeeded == 0 {
		return 0
	}
	return time.Duration(math.Round(float64(s.totalRTT) / float64(s.succeeded)))
}

func (s *tcpingStats) lossPercent() float64 {
	if s.sent == 0 {
		return 0
	}
	return float64(s.sent-s.succeeded) / float64(s.sent) * 100
}

func runTCPing(cmd *cobra.Command, args []string) error {
	host := args[0]
	port := args[1]
	addr := net.JoinHostPort(host, port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	fmt.Fprintf(cmd.OutOrStdout(), "TCPing %s: tcp connect ...\n", addr)

	stats := &tcpingStats{}
	seq := 0
	for {
		if tcpingCount > 0 && seq >= tcpingCount {
			break
		}

		if seq > 0 {
			select {
			case <-ctx.Done():
				printSummary(cmd, addr, stats)
				return nil
			case <-time.After(tcpingInterval):
			}
		}

		if ctx.Err() != nil {
			break
		}

		seq++
		start := time.Now()
		conn, err := net.DialTimeout("tcp", addr, tcpingTimeout)
		rtt := time.Since(start)

		if err != nil {
			stats.record(rtt, false)
			fmt.Fprintf(cmd.OutOrStdout(), "seq=%d %s - timeout/error: %v\n", seq, addr, err)
		} else {
			conn.Close()
			stats.record(rtt, true)
			fmt.Fprintf(cmd.OutOrStdout(), "seq=%d %s rtt=%v\n", seq, addr, rtt.Round(time.Microsecond))
		}
	}

	printSummary(cmd, addr, stats)
	return nil
}

func printSummary(cmd *cobra.Command, addr string, stats *tcpingStats) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintf(out, "--- %s tcping statistics ---\n", addr)
	fmt.Fprintf(out, "%d probes sent, %d successful, %.1f%% loss\n",
		stats.sent, stats.succeeded, stats.lossPercent())
	if stats.succeeded > 0 {
		fmt.Fprintf(out, "rtt min/avg/max = %v/%v/%v\n",
			stats.minRTT.Round(time.Microsecond),
			stats.avgRTT().Round(time.Microsecond),
			stats.maxRTT.Round(time.Microsecond))
	}
}
