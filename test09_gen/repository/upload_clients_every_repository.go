package repository

import (
	"errors"
	"gorm.io/gorm"
	"log/slog"
	"test09_gen/entity/model"
	"time"
)

// UploadClientsEveryRepository UploadClientsEvery 结构体持久层
// @author
// @date 2025-08-25T12:15:10.762172
type UploadClientsEveryRepository struct {
	db *gorm.DB
}

// NewUploadClientsEveryRepository 创建 UploadClientsEvery loadClientsEvery 持久层对象
func NewUploadClientsEveryRepository(db *gorm.DB) *UploadClientsEveryRepository {
	return &UploadClientsEveryRepository{
		db: db,
	}
}

// 由于有时需要开启事务，因此 DB *gorm.DB 可以选择从外部传入

// InsertUploadClientsEvery 新增UploadClientsEvery
func (ucer *UploadClientsEveryRepository) InsertUploadClientsEvery(uploadClientsEvery *model.UploadClientsEvery) (int, error) {
	slog.Info("UploadClientsEveryRepository.InsertUploadClientsEvery：", slog.Any("uploadClientsEvery", uploadClientsEvery))

	// 先查询是否有相同 name 的数据存在
	temp := &model.UploadClientsEvery{}
	// todo update name
	tx := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Where("hostname = ?", uploadClientsEvery.Hostname).First(temp)
	slog.Info("InsertUploadClientsEvery.Where Unique：", slog.Any("temp", temp))
	if !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return 0, errors.New("UploadClientsEveryRepository.InsertUploadClientsEvery.Where, 存在相同 name: " + temp.Hostname)
	}

	// 执行 Insert
	err := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Create(&uploadClientsEvery).Error

	if err != nil {
		return 0, errors.New("UploadClientsEveryRepository.InsertUploadClientsEvery.ucer.db.Create, 新增失败: " + err.Error())
	}
	return 1, nil
}

// BatchInsertUploadClientsEverys 批量新增UploadClientsEvery
func (ucer *UploadClientsEveryRepository) BatchInsertUploadClientsEverys(uploadClientsEverys []*model.UploadClientsEvery) (int, error) {
	slog.Info("UploadClientsEveryRepository.BatchInsertUploadClientsEverys：", slog.Any("uploadClientsEverys", uploadClientsEverys))

	result := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Create(&uploadClientsEverys)
	if result.Error != nil {
		return 0, errors.New("UploadClientsEveryRepository.BatchInsertUploadClientsEverys.Create, 新增失败: " + result.Error.Error())
	}
	return int(result.RowsAffected), nil
}

// UpdateUploadClientsEveryById 根据主键修改UploadClientsEvery的所有字段
func (ucer *UploadClientsEveryRepository) UpdateUploadClientsEveryById(uploadClientsEvery *model.UploadClientsEvery) (int, error) {
	slog.Info("UploadClientsEveryRepository.UpdateUploadClientsEveryById：", slog.Any("uploadClientsEvery", uploadClientsEvery))

	// 1、查询该Id是否存在
	if uploadClientsEvery.Id == 0 {
		return 0, errors.New("UploadClientsEveryRepository.UpdateUploadClientsEveryById Id 不能为空！！！: ")
	}

	// 2、再看看name是否重复
	temp := &model.UploadClientsEvery{}
	// todo update name
	tx := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Where("hostname = ?", uploadClientsEvery.Hostname).First(temp)
	slog.Info("UploadClientsEveryRepository.UpdateUploadClientsEveryById Unique：", slog.Any("temp", temp))
	if !errors.Is(tx.Error, gorm.ErrRecordNotFound) && temp.Id != uploadClientsEvery.Id {
		return 0, errors.New("UploadClientsEveryRepository.UpdateUploadClientsEveryById.Where, 存在相同 name: " + temp.Hostname)
	}

	// 3、执行修改
	//保存整个结构体（全字段更新）
	saveErr := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Save(uploadClientsEvery).Error
	if saveErr != nil {
		return 0, errors.New("UploadClientsEveryRepository.UpdateUploadClientsEveryById.Save, 修改失败: " + saveErr.Error())
	}
	return 1, nil
}

// UpdateUploadClientsEverySelective 修改UploadClientsEvery不为默认值的字段
func (ucer *UploadClientsEveryRepository) UpdateUploadClientsEverySelective(uploadClientsEvery *model.UploadClientsEvery) (int, error) {
	slog.Info("UploadClientsEveryRepository.UpdateUploadClientsEverySelective：", slog.Any("uploadClientsEvery", uploadClientsEvery))

	// ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Model().Updates()：只更新指定字段
	err := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Model(model.UploadClientsEvery{}).
		Where("id = ?", uploadClientsEvery.Id).Updates(uploadClientsEvery).Error
	if err != nil {
		return 0, errors.New("UploadClientsEveryRepository.UpdateUploadClientsEverySelective.Updates, 选择性修改失败: " + err.Error())
	}

	return 1, nil
}

// BatchUpdateUploadClientsEverySelective 批量修改UploadClientsEvery
func (ucer *UploadClientsEveryRepository) BatchUpdateUploadClientsEverySelective(uploadClientsEverys []model.UploadClientsEvery) error {
	return ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Transaction(func(tx *gorm.DB) error {
		for _, v := range uploadClientsEverys {
			err := tx.Model(&model.UploadClientsEvery{}).Where("id = ?", v.Id).Updates(v).Error
			if err != nil {
				return err // 触发回滚
			}
		}
		return nil // 提交事务
	})
}

// BatchDeleteUploadClientsEveryByState 批量软删除 UploadClientsEvery
func (ucer *UploadClientsEveryRepository) BatchDeleteUploadClientsEveryByState(idList []int64) error {
	slog.Info("UploadClientsEveryRepository.BatchDeleteUploadClientsEveryByState：", slog.Any("idList", idList))

	return ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Model(&model.UploadClientsEvery{}).
		Where("id IN ?", idList).
		Updates(map[string]any{
			"state":      0,
			"deleted_at": time.Now(),
		}).Error
}

// BatchDeleteUploadClientsEvery 根据主键批量删除 UploadClientsEvery
func (ucer *UploadClientsEveryRepository) BatchDeleteUploadClientsEvery(idList []int64) (int, error) {
	slog.Info("UploadClientsEveryRepository.BatchDeleteUploadClientsEvery：", slog.Any("idList", idList))

	// 当存在DeletedAt gorm.DeletedAt字段时为软删除，否则为物理删除
	result := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Where("id IN ?", idList).Delete(&model.UploadClientsEvery{})
	// result := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Model(&model.UploadClientsEvery{}).Where("id IN ?", idList).Update("state", 0)
	if result.Error != nil {
		return 0, errors.New("UploadClientsEveryRepository.BatchDeleteUploadClientsEvery.Delete, 删除失败: " + result.Error.Error())
	}

	//// 以下使用的是物理删除
	//result := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Unscoped().Delete(&model.UploadClientsEvery{}, "id in ?", idList)
	//if result.Error != nil {
	//	return 0, errors.New("UploadClientsEveryRepository.BatchDeleteUploadClientsEvery.Delete, 删除失败: " + result.Error.Error())
	//}

	return int(result.RowsAffected), nil
}

// FindUploadClientsEveryById 获取UploadClientsEvery详细信息
func (ucer *UploadClientsEveryRepository) FindUploadClientsEveryById(id int64) (*model.UploadClientsEvery, error) {
	slog.Info("UploadClientsEveryRepository.FindUploadClientsEveryById：", slog.Any("id", id))

	uploadClientsEvery := model.UploadClientsEvery{}
	err := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).First(&uploadClientsEvery, "id = ?", id).Error
	return &uploadClientsEvery, err
}

// FindUploadClientsEverysByIdList 根据主键批量查询UploadClientsEvery详细信息
func (ucer *UploadClientsEveryRepository) FindUploadClientsEverysByIdList(idList []int64) ([]*model.UploadClientsEvery, error) {
	slog.Info("UploadClientsEveryRepository.FindUploadClientsEverysByIdList：", slog.Any("idList", idList))

	var result []*model.UploadClientsEvery
	err := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Where("id IN ?", idList).Find(&result).Error
	return result, err
}

// FindUploadClientsEveryList 查询UploadClientsEvery列表
func (ucer *UploadClientsEveryRepository) FindUploadClientsEveryList(uploadClientsEvery model.UploadClientsEvery, startTime time.Time, endTime time.Time) ([]*model.UploadClientsEvery, error) {
	slog.Info("UploadClientsEveryRepository.FindUploadClientsEveryList：", slog.Any("uploadClientsEvery", uploadClientsEvery))

	var uploadClientsEverys []*model.UploadClientsEvery
	query := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Model(&model.UploadClientsEvery{})

	// 构造查询条件
	if uploadClientsEvery.Id != 0 {
		query = query.Where("id = ?", uploadClientsEvery.Id)
	}
	if uploadClientsEvery.Hostname != "" {
		query = query.Where("hostname LIKE ?", "%"+uploadClientsEvery.Hostname+"%")
	}
	if uploadClientsEvery.UniqueId != "" {
		query = query.Where("unique_id = ?", uploadClientsEvery.UniqueId)
	}
	if uploadClientsEvery.UploadSpeed != 0 {
		query = query.Where("upload_speed = ?", uploadClientsEvery.UploadSpeed)
	}
	if !uploadClientsEvery.Ctime.IsZero() {
		query = query.Where("ctime = ?", uploadClientsEvery.Ctime)
		// query = query.Where("DATE(ctime) = ?", uploadClientsEvery.$column.goField.Format("2006-01-02"))
	}

	if !startTime.IsZero() {
		query = query.Where("ctime >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("ctime <= ?", endTime)
	}

	// // 添加分页逻辑
	// if uploadClientsEvery.PageNum > 0 && uploadClientsEvery.PageSize > 0 {
	//     offset := (uploadClientsEvery.PageNum - 1) * uploadClientsEvery.PageSize
	//     query = query.Offset(offset).Limit(uploadClientsEvery.PageSize)
	// }

	err := query.Find(&uploadClientsEverys).Error
	return uploadClientsEverys, err
}

// FindUploadClientsEveryPageList 分页查询UploadClientsEvery列表
func (ucer *UploadClientsEveryRepository) FindUploadClientsEveryPageList(uploadClientsEvery model.UploadClientsEvery, startTime time.Time, endTime time.Time, pageNum int, pageSize int) ([]*model.UploadClientsEvery, int64, error) {
	slog.Info("UploadClientsEveryRepository.FindUploadClientsEveryPageList：", slog.Any("uploadClientsEvery", uploadClientsEvery))

	var (
		uploadClientsEverys []*model.UploadClientsEvery
		total               int64
	)

	query := ucer.db.Table((&model.UploadClientsEvery{}).TableName()).Model(&model.UploadClientsEvery{})

	// 构造查询条件
	if uploadClientsEvery.Id != 0 {
		query = query.Where("id = ?", uploadClientsEvery.Id)
	}
	if uploadClientsEvery.Hostname != "" {
		query = query.Where("hostname LIKE ?", "%"+uploadClientsEvery.Hostname+"%")
	}
	if uploadClientsEvery.UniqueId != "" {
		query = query.Where("unique_id = ?", uploadClientsEvery.UniqueId)
	}
	if uploadClientsEvery.UploadSpeed != 0 {
		query = query.Where("upload_speed = ?", uploadClientsEvery.UploadSpeed)
	}
	if !uploadClientsEvery.Ctime.IsZero() {
		query = query.Where("ctime = ?", uploadClientsEvery.Ctime)
		// query = query.Where("DATE(ctime) = ?", uploadClientsEvery.$column.goField.Format("2006-01-02"))
	}

	if !startTime.IsZero() {
		query = query.Where("ctime >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("ctime <= ?", endTime)
	}

	// 分页参数默认值
	if pageNum <= 0 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 分页数据
	// todo update ctime
	err := query.
		Count(&total).Order("ctime desc").
		Limit(pageSize).Offset((pageNum - 1) * pageSize).
		Order("ctime desc").
		Find(&uploadClientsEverys).Error

	if err != nil {
		return nil, 0, err
	}

	return uploadClientsEverys, total, nil
}
