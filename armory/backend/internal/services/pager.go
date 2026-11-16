package services

type Pager struct {
	Page    int
	Pages   int
	Total   int
	First   int
	Last    int
	HasPrev bool
	HasNext bool
	Prev    int
	Next    int
	size    int
}

func newPager(total, size, page int) Pager {
	pages := max((total+size-1)/size, 1)
	page = min(max(page, 1), pages)
	return Pager{
		Page: page, Pages: pages, Total: total, size: size,
		HasPrev: page > 1, HasNext: page < pages, Prev: page - 1, Next: page + 1,
	}
}

func (p Pager) offset() int {
	return (p.Page - 1) * p.size
}

func (p *Pager) shown(n int) {
	if p.Total > 0 {
		p.First = p.offset() + 1
		p.Last = p.First + n - 1
	}
}
