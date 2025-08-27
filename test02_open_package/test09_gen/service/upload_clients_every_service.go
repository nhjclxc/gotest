package service

import (
	"gorm.io/gorm"
	"test09_gen/entity/model"
	"test09_gen/entity/req"
	"test09_gen/repository"
	commonUtils "test09_gen/utils"
)

// UploadClientsEveryService UploadClientsEvery Service 层
type UploadClientsEveryService struct {
	db                           *gorm.DB
	uploadClientsEveryRepository *repository.UploadClientsEveryRepository
}

// NewUploadClientsEveryService 创建 UploadClientsEverploadClientsEvery 业务层对象
func NewUploadClientsEveryService(db *gorm.DB, uploadClientsEveryRepository *repository.UploadClientsEveryRepository) *UploadClientsEveryService {
	return &UploadClientsEveryService{
		db:                           db,
		uploadClientsEveryRepository: uploadClientsEveryRepository,
	}
}

// InsertUploadClientsEvery 新增UploadClientsEvery
func (uces *UploadClientsEveryService) InsertUploadClientsEvery(uploadClientsEvery model.UploadClientsEvery) (int, error) {

	return uces.uploadClientsEveryRepository.InsertUploadClientsEvery(&uploadClientsEvery)
}

// UpdateUploadClientsEvery 修改UploadClientsEvery
func (uces *UploadClientsEveryService) UpdateUploadClientsEvery(uploadClientsEvery model.UploadClientsEvery) (int, error) {

	return uces.uploadClientsEveryRepository.UpdateUploadClientsEveryById(&uploadClientsEvery)
}

// DeleteUploadClientsEvery 删除UploadClientsEvery
func (uces *UploadClientsEveryService) DeleteUploadClientsEvery(idList []int64) (int, error) {

	return uces.uploadClientsEveryRepository.BatchDeleteUploadClientsEvery(idList)
}

// GetUploadClientsEveryById 获取UploadClientsEvery业务详细信息
func (uces *UploadClientsEveryService) GetUploadClientsEveryById(id int64) (*model.UploadClientsEvery, error) {

	uploadClientsEvery, err := uces.uploadClientsEveryRepository.FindUploadClientsEveryById(id)
	if err != nil {
		return nil, err
	}

	return uploadClientsEvery, nil
}

// GetUploadClientsEveryList 查询UploadClientsEvery业务列表
func (uces *UploadClientsEveryService) GetUploadClientsEveryList(uploadClientsEveryReq req.UploadClientsEveryReq) (res any, err error) {

	uploadClientsEvery, err := uploadClientsEveryReq.ReqToModel()
	uploadClientsEveryList, err := uces.uploadClientsEveryRepository.FindUploadClientsEveryList(*uploadClientsEvery, uploadClientsEveryReq.SatrtTime, uploadClientsEveryReq.EndTime)
	if err != nil {
		return nil, err
	}

	return uploadClientsEveryList, nil
}

// GetUploadClientsEveryPageList 分页查询UploadClientsEvery业务列表
func (uces *UploadClientsEveryService) GetUploadClientsEveryPageList(uploadClientsEveryReq req.UploadClientsEveryReq) (res any, err error) {

	uploadClientsEvery, err := uploadClientsEveryReq.ReqToModel()
	uploadClientsEveryList, total, err := uces.uploadClientsEveryRepository.FindUploadClientsEveryPageList(*uploadClientsEvery, uploadClientsEveryReq.SatrtTime, uploadClientsEveryReq.EndTime, uploadClientsEveryReq.PageNum, uploadClientsEveryReq.PageSize)
	if err != nil {
		return nil, err
	}

	return commonUtils.BuildPageData[model.UploadClientsEvery](uploadClientsEveryList, total, uploadClientsEveryReq.PageNum, uploadClientsEveryReq.PageSize), nil
}

// ExportUploadClientsEvery 导出UploadClientsEvery业务列表
func (uces *UploadClientsEveryService) ExportUploadClientsEvery(uploadClientsEveryReq req.UploadClientsEveryReq) (res any, err error) {

	uploadClientsEvery, err := uploadClientsEveryReq.ReqToModel()
	uces.uploadClientsEveryRepository.FindUploadClientsEveryPageList(*uploadClientsEvery, uploadClientsEveryReq.SatrtTime, uploadClientsEveryReq.EndTime, 1, 10000)
	// 实现导出 ...

	return nil, nil
}
