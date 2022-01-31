package types

import "testing"

func TestService(t *testing.T) {

	p := APT{}
	isService := p.Service()
	expected := false
	if isService {
		t.Errorf("expected %v and got %v", expected, isService)
	}

	p.Status = StatusRestarted
	expected = true
	if isService {
		t.Errorf("expected %v and got %v", expected, isService)
	}

	p.Status = StatusStarted
	expected = true
	if isService {
		t.Errorf("expected %v and got %v", expected, isService)
	}

	p.Status = StatusNotInstalled
	expected = false
	if isService {
		t.Errorf("expected %v and got %v", expected, isService)
	}
}
