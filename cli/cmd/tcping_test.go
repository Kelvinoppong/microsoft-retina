// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.
package cmd

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runTCPingForTest(t *testing.T, args []string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	Retina.SetOut(&buf)
	Retina.SetErr(&buf)
	Retina.SetArgs(append([]string{"tcping"}, args...))
	t.Cleanup(func() {
		Retina.SetArgs(nil)
		Retina.SetOut(nil)
		Retina.SetErr(nil)
		tcpingCount = 0
		tcpingInterval = 1 * time.Second
		tcpingTimeout = 2 * time.Second
	})
	err := Retina.Execute()
	return buf.String(), err
}

func TestTCPingSuccess(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)

	output, err := runTCPingForTest(t, []string{"127.0.0.1", port, "--count", "3", "--interval", "50ms"})
	require.NoError(t, err)

	assert.Contains(t, output, "TCPing 127.0.0.1:")
	assert.Contains(t, output, "seq=1")
	assert.Contains(t, output, "seq=3")
	assert.Contains(t, output, "3 probes sent, 3 successful, 0.0% loss")
	assert.Contains(t, output, "rtt min/avg/max")
}

func TestTCPingFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	_, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	ln.Close()

	output, err := runTCPingForTest(t, []string{"127.0.0.1", port, "--count", "2", "--interval", "50ms", "--timeout", "200ms"})
	require.NoError(t, err)

	assert.Contains(t, output, "2 probes sent, 0 successful, 100.0% loss")
}

func TestTCPingStats(t *testing.T) {
	s := &tcpingStats{}

	s.record(10*time.Millisecond, true)
	s.record(20*time.Millisecond, true)
	s.record(30*time.Millisecond, false)
	s.record(5*time.Millisecond, true)

	assert.Equal(t, 4, s.sent)
	assert.Equal(t, 3, s.succeeded)
	assert.Equal(t, 5*time.Millisecond, s.minRTT)
	assert.Equal(t, 20*time.Millisecond, s.maxRTT)
	assert.InDelta(t, 25.0, s.lossPercent(), 0.1)

	avg := s.avgRTT()
	expected := (10*time.Millisecond + 20*time.Millisecond + 5*time.Millisecond) / 3
	assert.InDelta(t, float64(expected), float64(avg), float64(time.Millisecond))
}

func TestTCPingHelp(t *testing.T) {
	output, err := runTCPingForTest(t, []string{"--help"})
	require.NoError(t, err)

	assert.Contains(t, output, "tcping HOST PORT")
	assert.Contains(t, output, "Probe a TCP port by performing a TCP handshake")
	assert.Contains(t, output, "--count")
	assert.Contains(t, output, "--interval")
	assert.Contains(t, output, "--timeout")
}

func TestTCPingMissingArgs(t *testing.T) {
	err := tcpingCmd.Args(tcpingCmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")

	err = tcpingCmd.Args(tcpingCmd, []string{"only-host"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")

	err = tcpingCmd.Args(tcpingCmd, []string{"host", "port"})
	require.NoError(t, err)
}

func TestTCPingStatsEmpty(t *testing.T) {
	s := &tcpingStats{}
	assert.Equal(t, time.Duration(0), s.avgRTT())
	assert.Equal(t, float64(0), s.lossPercent())
}

func TestTCPingStatsAllFail(t *testing.T) {
	s := &tcpingStats{}
	s.record(10*time.Millisecond, false)
	s.record(10*time.Millisecond, false)

	assert.Equal(t, 2, s.sent)
	assert.Equal(t, 0, s.succeeded)
	assert.Equal(t, 100.0, s.lossPercent())
	assert.Equal(t, time.Duration(0), s.avgRTT())
}

func TestPrintSummary(t *testing.T) {
	s := &tcpingStats{}
	s.record(10*time.Millisecond, true)
	s.record(20*time.Millisecond, true)

	var buf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&buf)

	printSummary(c, "example.com:80", s)
	output := buf.String()
	assert.Contains(t, output, "example.com:80 tcping statistics")
	assert.Contains(t, output, fmt.Sprintf("2 probes sent, 2 successful, %.1f%% loss", 0.0))
	assert.Contains(t, output, "rtt min/avg/max")
}
