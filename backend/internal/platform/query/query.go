package query

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
)

type Page struct {
	Number int
	Size   int
	Sort   string
	Order  string
	Search string
}

func Parse(c *gin.Context) Page {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page < 1 {
		page = 1
	}
	size, err := strconv.Atoi(c.Query("page_size"))
	if err != nil || size < 1 || size > 200 {
		size = 20
	}
	sortField := c.Query("sort")
	if sortField == "" {
		sortField = "created_at"
	}
	allowedSort := map[string]bool{
		"created_at": true, "updated_at": true, "name": true, "email": true, "code": true, "status": true,
	}
	if !allowedSort[sortField] {
		sortField = "created_at"
	}
	order := strings.ToLower(c.Query("order"))
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	return Page{Number: page, Size: size, Sort: sortField, Order: order, Search: strings.TrimSpace(c.Query("search"))}
}

func (p Page) Offset() int { return (p.Number - 1) * p.Size }

func Meta(page Page, total int64) response.PageMeta {
	totalPages := int((total + int64(page.Size) - 1) / int64(page.Size))
	return response.PageMeta{Page: page.Number, PageSize: page.Size, Total: total, TotalPages: totalPages}
}
