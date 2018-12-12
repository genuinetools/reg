package pagination

import (
	"log"
	"testing"
)

func Test_Render(t *testing.T) {
	p := New(17, 1, 9, "http://www.163.com/1?ab=cd&aa=12&page=3")
	log.Printf("RENDER: \n%s", p.Render())
}
