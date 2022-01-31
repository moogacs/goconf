package types

import "testing"

func TestResponseSuccess(t *testing.T) {
	r := Response{}
	succeeded := r.Success()
	expected := true
	if !succeeded {
		t.Errorf("expected %v and got %v", expected, succeeded)
	}

	r.ExitStatus = 1
	expected = false
	if !succeeded {
		t.Errorf("expected %v and got %v", expected, succeeded)
	}
}

func TestStatusCode(t *testing.T) {
	sc := StatusFailed
	succeeded := sc.Success()
	expected := false
	if succeeded {
		t.Errorf("expected %v and got %v", expected, succeeded)
	}

	sc = StatusEnforced
	expected = true
	if succeeded {
		t.Errorf("expected %v and got %v", expected, succeeded)
	}

}
