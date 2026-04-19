package pagination

import (
	"math"

	"github.com/gin-gonic/gin"
)

type Params struct {
	Limit  int `form:"limit"`
	Offset int `form:"offset"`
}

type Page[T any] struct {
	Items  []T   `json:"items"`
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Pages  int   `json:"pages"`
}

func ParseParams(ctx *gin.Context) Params {
	var p Params
	//nolint:errcheck // Intentionally ignoring binding error; defaults will be applied below
	_ = ctx.ShouldBindQuery(&p)
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 25
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}

func NewPage[T any](items []T, total int64, p Params) Page[T] {
	pages := 0
	if p.Limit > 0 {
		pages = int(math.Ceil(float64(total) / float64(p.Limit)))
	}
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items, Total: total, Limit: p.Limit, Offset: p.Offset, Pages: pages}
}
