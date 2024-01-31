package collector

import (
	"fmt"
	"regexp"
	"testing"
)

func TestTTT(t *testing.T) {
	r := regexp.MustCompile("^(cni|docker|kube-ipvs|dummy)[0-9]+|veth.*|lo")
	fmt.Println(r.MatchString("lo"))
}
