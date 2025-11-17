package endpoint

import (
	"testing"
)

func TestParseEndpointLocal(t *testing.T) {
	ep, err := ParseEndpoint("./data", 22, SSHOptions{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ep.Type != EndpointLocal {
		t.Fatalf("expected local endpoint")
	}
	if ep.Path != "data" {
		t.Fatalf("unexpected path %s", ep.Path)
	}
}

func TestParseEndpointRemote(t *testing.T) {
	ep, err := ParseEndpoint("user@example.com:/var/data", 2222, SSHOptions{Identity: "~/.ssh/id"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ep.Type != EndpointRemote {
		t.Fatalf("expected remote endpoint")
	}
	if ep.User != "user" || ep.Host != "example.com" {
		t.Fatalf("unexpected remote info: %+v", ep)
	}
	if ep.SSHOpts.Port != 2222 {
		t.Fatalf("unexpected port %d", ep.SSHOpts.Port)
	}
	if ep.SSHOpts.Identity == "" {
		t.Fatalf("identity missing")
	}
}
