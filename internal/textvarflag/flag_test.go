package textvarflag_test

import (
	"flag"
	"net"
	"testing"

	"github.com/ossf/criticality_score/internal/textvarflag"
	"github.com/sirupsen/logrus"
)

var defaultIP = net.IPv4(192, 168, 0, 100)

func TestFlagUnset(t *testing.T) {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	var ip net.IP
	textvarflag.TextVar(fs, &ip, "ip", defaultIP, "usage")
	err := fs.Parse([]string{"arg"})
	if err != nil {
		t.Fatalf("Parse() == %v, want nil", err)
	}
	if !defaultIP.Equal(ip) {
		t.Fatalf("ip == %v, want %v", ip, defaultIP)
	}
}

func TestFlagSet(t *testing.T) {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	var ip net.IP
	textvarflag.TextVar(fs, &ip, "ip", defaultIP, "usage")
	err := fs.Parse([]string{"-ip=127.0.0.1", "arg"})
	if err != nil {
		t.Fatalf("Parse() == %v, want nil", err)
	}
	if expect := net.IPv4(127, 0, 0, 1); !expect.Equal(ip) {
		t.Fatalf("ip == %v, want %v", ip, logrus.FatalLevel)
	}
}

func TestFlagSetError(t *testing.T) {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	var ip net.IP
	textvarflag.TextVar(fs, &ip, "ip", defaultIP, "usage")
	err := fs.Parse([]string{"-ip=256.0.0.1", "arg"})
	if err == nil {
		t.Fatalf("Parse() == nil, want an error")
	}
}
