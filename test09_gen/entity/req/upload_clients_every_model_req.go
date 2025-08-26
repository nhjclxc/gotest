package req

import (
	"fmt"
	"github.com/jinzhu/copier"
	"test09_gen/entity/model"
	"time"
)

// UploadClientsEveryReq UploadClientsEvery对象请求结构体
// @author
// @date 2025-08-22T10:34:07.338521
type UploadClientsEveryReq struct {
	model.UploadClientsEvery

	Keyword string `form:"keyword"` // 模糊搜索字段

	PageNum  int `form:"pageNum"`  // 页码
	PageSize int `form:"pageSize"` // 页大小

	SatrtTime time.Time `form:"satrtTime" time_format:"2006-01-02 15:04:05"` // 开始时间
	EndTime   time.Time `form:"endTime" time_format:"2006-01-02 15:04:05"`   // 结束时间
}

// ReqToModel modelReq 转化为 model
func (uce *UploadClientsEveryReq) ReqToModel() (uploadClientsEvery *model.UploadClientsEvery, err error) {
	// go get github.com/jinzhu/copier

	uploadClientsEvery = &model.UploadClientsEvery{} // copier.Copy 不会自动为其分配空间，所以初始化指针指向的结构体
	err = copier.Copy(&uploadClientsEvery, &uce)
	return
}

// ModelToReq model 转化为 modelReq
func (uce *UploadClientsEveryReq) ModelToReq(uploadClientsEvery *model.UploadClientsEvery) error {
	// go get github.com/jinzhu/copier

	err := copier.Copy(&uce, &uploadClientsEvery)
	if err != nil {
		fmt.Printf("ReqTo Copy error: %v", err)
		return err
	}
	return nil
}
