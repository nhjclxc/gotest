package resp

import (
	"fmt"
	"github.com/jinzhu/copier"
	"test09_gen/entity/model"
)

// UploadClientsEveryResp UploadClientsEvery对象响应结构体
// @author
// @date 2025-08-22T10:34:07.338521
type UploadClientsEveryResp struct {
	model.UploadClientsEvery

	Foo string `form:"foo"` // foo
	Bar string `form:"bar"` // bar
	// ...
}

// ModelToResp model 转化为 modelResp
func (uce *UploadClientsEveryResp) ModelToResp(uploadClientsEvery *model.UploadClientsEvery) error {
	// go get github.com/jinzhu/copier

	if uploadClientsEvery == nil {
		uploadClientsEvery = &model.UploadClientsEvery{} // copier.Copy 不会自动为其分配空间，所以初始化指针指向的结构体
	}
	err := copier.Copy(&uce, &uploadClientsEvery)
	if err != nil {
		fmt.Printf("ModelToResp Copy error: %v", err)
		return err
	}
	return nil
}
