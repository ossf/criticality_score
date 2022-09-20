// Copyright 2022 Criticality Score Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package textvarflag_test

import (
	"flag"
	"net"
	"testing"

	"github.com/ossf/criticality_score/internal/textvarflag"
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
		t.Fatalf("ip == %v, want %v", ip, expect)
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
