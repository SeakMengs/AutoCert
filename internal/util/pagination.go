package util

import "github.com/SeakMengs/AutoCert/internal/constant"

func CalculateTotalPage(totalItems int64, pageSize uint) int {
	if pageSize <= 0 {
		pageSize = constant.DefaultPageSize
	}
	if totalItems == 0 {
		return 1
	}
	totalPage := int(totalItems / int64(pageSize))
	if totalItems%int64(pageSize) != 0 {
		totalPage++
	}
	return totalPage
}
