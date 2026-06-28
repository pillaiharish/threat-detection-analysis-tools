package main

import (
	"reflect"
	"testing"
)

func TestParsePorts_SinglePort(t *testing.T) {
	got, err := parsePorts("80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{80}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParsePorts_Range(t *testing.T) {
	got, err := parsePorts("1-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{1, 2, 3, 4, 5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParsePorts_MixedRangesAndLists(t *testing.T) {
	got, err := parsePorts("1-5,80,443,1000-1002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{1, 2, 3, 4, 5, 80, 443, 1000, 1001, 1002}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParsePorts_DedupAndSort(t *testing.T) {
	got, err := parsePorts("80,1-5,80,3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{1, 2, 3, 4, 5, 80}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParsePorts_Port0(t *testing.T) {
	_, err := parsePorts("0")
	if err == nil {
		t.Fatal("expected error for port 0")
	}
}

func TestParsePorts_OutOfRange(t *testing.T) {
	_, err := parsePorts("70000")
	if err == nil {
		t.Fatal("expected error for port 70000")
	}
}

func TestParsePorts_InvertedRange(t *testing.T) {
	_, err := parsePorts("5-1")
	if err == nil {
		t.Fatal("expected error for inverted range")
	}
}

func TestParsePorts_InvalidSpec(t *testing.T) {
	_, err := parsePorts("abc")
	if err == nil {
		t.Fatal("expected error for non-numeric port")
	}
}

func TestParsePorts_EmptySpec(t *testing.T) {
	got, err := parsePorts("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestParseTargets_SingleIP(t *testing.T) {
	got, err := parseTargets("192.168.1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"192.168.1.1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseTargets_MultipleIPsDeduped(t *testing.T) {
	got, err := parseTargets("192.168.1.1,192.168.1.1,10.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"10.0.0.1", "192.168.1.1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseTargets_CIDR(t *testing.T) {
	got, err := parseTargets("192.168.1.0/30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"192.168.1.0", "192.168.1.1", "192.168.1.2", "192.168.1.3"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseTargets_CIDRLimitExceeded(t *testing.T) {
	_, err := parseTargets("10.0.0.0/8")
	if err == nil {
		t.Fatal("expected error for /8 CIDR (exceeds maxCIDRHosts)")
	}
}

func TestParseTargets_InvalidCIDR(t *testing.T) {
	_, err := parseTargets("192.168.1.0/33")
	if err == nil {
		t.Fatal("expected error for invalid CIDR")
	}
}

func TestParseTargets_UnresolvableHost(t *testing.T) {
	_, err := parseTargets("this-host-does-not-exist.invalid")
	if err == nil {
		t.Fatal("expected error for unresolvable hostname")
	}
}

func TestParseTargets_EmptySpec(t *testing.T) {
	got, err := parseTargets(",,")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestValidPort(t *testing.T) {
	tests := []struct {
		port int
		want bool
	}{
		{0, false},
		{1, true},
		{65535, true},
		{65536, false},
		{-1, false},
	}
	for _, tt := range tests {
		if got := validPort(tt.port); got != tt.want {
			t.Errorf("validPort(%d) = %v, want %v", tt.port, got, tt.want)
		}
	}
}