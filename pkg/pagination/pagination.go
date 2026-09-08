package pagination

// PageParam 分页请求参数
type PageParam struct {
	PageNo   int `form:"pageNo,default=1" json:"pageNo"`
	PageSize int `form:"pageSize,default=10" json:"pageSize" binding:"max=200"`
}

// PageResult 分页返回结果
type PageResult[T any] struct {
	List  []T   `json:"list"`
	Total int64 `json:"total"`
}

func (p *PageParam) GetOffset() int {
	if p.PageNo < 1 {
		p.PageNo = 1
	}
	p.GetLimit()
	if p.PageSize == -1 {
		return 0
	}
	return (p.PageNo - 1) * p.PageSize
}

func (p *PageParam) GetLimit() int {
	if p.PageSize == -1 {
		return -1
	}
	if p.PageSize > 200 {
		p.PageSize = 200
	}
	if p.PageSize < 1 {
		p.PageSize = 10
	}
	return p.PageSize
}

func NewPageResult[T any](list []T, total int64) *PageResult[T] {
	return &PageResult[T]{
		List:  list,
		Total: total,
	}
}

func NewEmptyPageResult[T any]() *PageResult[T] {
	return &PageResult[T]{
		List:  make([]T, 0),
		Total: 0,
	}
}
