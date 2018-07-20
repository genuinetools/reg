package pagination

import (
	"math"
)

type Pagination struct {
	perPage     int
	totalAmount int
	currentPage int
	totalPage   int
	baseUrl     string

	// render parts
	firstPart   []string
	middlePart  []string
	lastPart    []string
}

// constructor
func New(totalAmount, perPage, currentPage int, baseUrl string) *Pagination {
	if currentPage == 0 {
		currentPage = 1
	}

	n := int(math.Ceil(float64(totalAmount) / float64(perPage)))
	if currentPage > n {
		currentPage = n
	}

	return &Pagination{
		perPage:     perPage,
		totalAmount: totalAmount,
		currentPage: currentPage,
		totalPage:int(math.Ceil(float64(totalAmount) / float64(perPage))),
		baseUrl: baseUrl,
	}
}

// 总共几页
func (p *Pagination) TotalPages() int {
	return p.totalPage
}

// 是否要显示分页
func (p *Pagination) HasPages() bool {
	return p.TotalPages() > 1
}
