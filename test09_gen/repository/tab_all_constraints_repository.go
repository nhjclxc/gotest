package repository

import (
	"errors"
	"gorm.io/gorm"
	"log/slog"
	"test09_gen/entity/model"
	"time"
)

// TabAllConstraintsRepository 所有约束情况示例  结构体持久层
// @author
// @date 2025-08-25T16:18:49.753
type TabAllConstraintsRepository struct {
	db *gorm.DB
}

// NewTabAllConstraintsRepository 创建 TabAllConstraints 所有约束情况示例  持久层对象
func NewTabAllConstraintsRepository(db *gorm.DB) *TabAllConstraintsRepository {
	return &TabAllConstraintsRepository{
		db: db,
	}
}

// 由于有时需要开启事务，因此 DB *gorm.DB 可以选择从外部传入

// InsertTabAllConstraints 新增所有约束情况示例
func (tacr *TabAllConstraintsRepository) InsertTabAllConstraints(tabAllConstraints *model.TabAllConstraints) (int, error) {
	slog.Info("TabAllConstraintsRepository.InsertTabAllConstraints：", slog.Any("tabAllConstraints", tabAllConstraints))

	// 先查询是否有相同 name 的数据存在
	temp := &model.TabAllConstraints{}
	// todo update name
	tx := tacr.db.Table((&model.TabAllConstraints{}).TableName()).Where("name = ?", tabAllConstraints.Username).First(temp)
	slog.Info("InsertTabAllConstraints.Where Unique：", slog.Any("temp", temp))
	if !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return 0, errors.New("TabAllConstraintsRepository.InsertTabAllConstraints.Where, 存在相同 name: " + temp.Username)
	}

	// 执行 Insert
	err := tacr.db.Table((&model.TabAllConstraints{}).TableName()).Create(&tabAllConstraints).Error

	if err != nil {
		return 0, errors.New("TabAllConstraintsRepository.InsertTabAllConstraints.tacr.db.Create, 新增失败: " + err.Error())
	}
	return 1, nil
}

// BatchInsertTabAllConstraintss 批量新增所有约束情况示例
func (tacr *TabAllConstraintsRepository) BatchInsertTabAllConstraintss(tabAllConstraintss []*model.TabAllConstraints) (int, error) {
	slog.Info("TabAllConstraintsRepository.BatchInsertTabAllConstraintss：", slog.Any("tabAllConstraintss", tabAllConstraintss))

	result := tacr.db.Table((&model.TabAllConstraints{}).TableName()).Create(&tabAllConstraintss)
	if result.Error != nil {
		return 0, errors.New("TabAllConstraintsRepository.BatchInsertTabAllConstraintss.Create, 新增失败: " + result.Error.Error())
	}
	return int(result.RowsAffected), nil
}

// UpdateTabAllConstraintsById 根据主键修改所有约束情况示例 的所有字段
func (tacr *TabAllConstraintsRepository) UpdateTabAllConstraintsById(tabAllConstraints *model.TabAllConstraints) (int, error) {
	slog.Info("TabAllConstraintsRepository.UpdateTabAllConstraintsById：", slog.Any("tabAllConstraints", tabAllConstraints))

	// 1、查询该Id是否存在
	if tabAllConstraints.Id == 0 {
		return 0, errors.New("TabAllConstraintsRepository.UpdateTabAllConstraintsById Id 不能为空！！！: ")
	}

	// 2、再看看name是否重复
	temp := &model.TabAllConstraints{}
	// todo update name
	tx := tacr.db.Table((&model.TabAllConstraints{}).TableName()).Where("name = ?", tabAllConstraints.Username).First(temp)
	slog.Info("TabAllConstraintsRepository.UpdateTabAllConstraintsById Unique：", slog.Any("temp", temp))
	if !errors.Is(tx.Error, gorm.ErrRecordNotFound) && temp.Id != tabAllConstraints.Id {
		return 0, errors.New("TabAllConstraintsRepository.UpdateTabAllConstraintsById.Where, 存在相同 name: " + temp.Username)
	}

	// 3、执行修改
	//保存整个结构体（全字段更新）
	saveErr := tacr.db.Table((&model.TabAllConstraints{}).TableName()).Save(tabAllConstraints).Error
	if saveErr != nil {
		return 0, errors.New("TabAllConstraintsRepository.UpdateTabAllConstraintsById.Save, 修改失败: " + saveErr.Error())
	}
	return 1, nil
}

// UpdateTabAllConstraintsSelective 修改所有约束情况示例 不为默认值的字段
func (tacr *TabAllConstraintsRepository) UpdateTabAllConstraintsSelective(tabAllConstraints *model.TabAllConstraints) (int, error) {
	slog.Info("TabAllConstraintsRepository.UpdateTabAllConstraintsSelective：", slog.Any("tabAllConstraints", tabAllConstraints))

	// tacr.db.Table((&model.TabAllConstraints{}).TableName()).Model().Updates()：只更新指定字段
	err := tacr.db.Table((&model.TabAllConstraints{}).TableName()).Model(model.TabAllConstraints{}).
		Where("id = ?", tabAllConstraints.Id).Updates(tabAllConstraints).Error
	if err != nil {
		return 0, errors.New("TabAllConstraintsRepository.UpdateTabAllConstraintsSelective.Updates, 选择性修改失败: " + err.Error())
	}

	return 1, nil
}

// BatchUpdateTabAllConstraintsSelective 批量修改所有约束情况示例
func (tacr *TabAllConstraintsRepository) BatchUpdateTabAllConstraintsSelective(tabAllConstraintss []model.TabAllConstraints) error {
	return tacr.db.Table((&model.TabAllConstraints{}).TableName()).Transaction(func(tx *gorm.DB) error {
		for _, v := range tabAllConstraintss {
			err := tx.Model(&model.TabAllConstraints{}).Where("id = ?", v.Id).Updates(v).Error
			if err != nil {
				return err // 触发回滚
			}
		}
		return nil // 提交事务
	})
}

// BatchDeleteTabAllConstraintsByState 批量软删除 所有约束情况示例
func (tacr *TabAllConstraintsRepository) BatchDeleteTabAllConstraintsByState(idList []int64) error {
	slog.Info("TabAllConstraintsRepository.BatchDeleteTabAllConstraintsByState：", slog.Any("idList", idList))

	return tacr.db.Table((&model.TabAllConstraints{}).TableName()).Model(&model.TabAllConstraints{}).
		Where("id IN ?", idList).
		Updates(map[string]any{
			"state":      0,
			"deleted_at": time.Now(),
		}).Error
}

// BatchDeleteTabAllConstraints 根据主键批量删除 所有约束情况示例
func (tacr *TabAllConstraintsRepository) BatchDeleteTabAllConstraints(idList []int64) (int, error) {
	slog.Info("TabAllConstraintsRepository.BatchDeleteTabAllConstraints：", slog.Any("idList", idList))

	// 当存在DeletedAt gorm.DeletedAt字段时为软删除，否则为物理删除
	result := tacr.db.Table((&model.TabAllConstraints{}).TableName()).Where("id IN ?", idList).Delete(&model.TabAllConstraints{})
	// result := tacr.db.Table((&model.TabAllConstraints{}).TableName()).Model(&model.TabAllConstraints{}).Where("id IN ?", idList).Update("state", 0)
	if result.Error != nil {
		return 0, errors.New("TabAllConstraintsRepository.BatchDeleteTabAllConstraints.Delete, 删除失败: " + result.Error.Error())
	}

	//// 以下使用的是物理删除
	//result := tacr.db.Table((&model.TabAllConstraints{}).TableName()).Unscoped().Delete(&model.TabAllConstraints{}, "id in ?", idList)
	//if result.Error != nil {
	//	return 0, errors.New("TabAllConstraintsRepository.BatchDeleteTabAllConstraints.Delete, 删除失败: " + result.Error.Error())
	//}

	return int(result.RowsAffected), nil
}

// FindTabAllConstraintsById 获取所有约束情况示例 详细信息
func (tacr *TabAllConstraintsRepository) FindTabAllConstraintsById(id int64) (*model.TabAllConstraints, error) {
	slog.Info("TabAllConstraintsRepository.FindTabAllConstraintsById：", slog.Any("id", id))

	tabAllConstraints := model.TabAllConstraints{}
	err := tacr.db.Table((&model.TabAllConstraints{}).TableName()).First(&tabAllConstraints, "id = ?", id).Error
	return &tabAllConstraints, err
}

// FindTabAllConstraintssByIdList 根据主键批量查询所有约束情况示例 详细信息
func (tacr *TabAllConstraintsRepository) FindTabAllConstraintssByIdList(idList []int64) ([]*model.TabAllConstraints, error) {
	slog.Info("TabAllConstraintsRepository.FindTabAllConstraintssByIdList：", slog.Any("idList", idList))

	var result []*model.TabAllConstraints
	err := tacr.db.Table((&model.TabAllConstraints{}).TableName()).Where("id IN ?", idList).Find(&result).Error
	return result, err
}

// FindTabAllConstraintsList 查询所有约束情况示例 列表
func (tacr *TabAllConstraintsRepository) FindTabAllConstraintsList(tabAllConstraints model.TabAllConstraints, startTime time.Time, endTime time.Time) ([]*model.TabAllConstraints, error) {
	slog.Info("TabAllConstraintsRepository.FindTabAllConstraintsList：", slog.Any("tabAllConstraints", tabAllConstraints))

	var tabAllConstraintss []*model.TabAllConstraints
	query := tacr.db.Table((&model.TabAllConstraints{}).TableName()).Model(&model.TabAllConstraints{})

	// 构造查询条件
	if tabAllConstraints.Id != 0 {
		query = query.Where("id = ?", tabAllConstraints.Id)
	}
	if tabAllConstraints.Username != "" {
		query = query.Where("username LIKE ?", "%"+tabAllConstraints.Username+"%")
	}
	if tabAllConstraints.Email != "" {
		query = query.Where("email = ?", tabAllConstraints.Email)
	}
	if tabAllConstraints.Phone != "" {
		query = query.Where("phone = ?", tabAllConstraints.Phone)
	}
	if tabAllConstraints.CountryCode != "" {
		query = query.Where("country_code = ?", tabAllConstraints.CountryCode)
	}
	if tabAllConstraints.Status != 0 {
		query = query.Where("status = ?", tabAllConstraints.Status)
	}
	if tabAllConstraints.Role != "" {
		query = query.Where("role = ?", tabAllConstraints.Role)
	}
	if tabAllConstraints.Age != 0 {
		query = query.Where("age = ?", tabAllConstraints.Age)
	}
	if tabAllConstraints.Balance != 0 {
		query = query.Where("balance = ?", tabAllConstraints.Balance)
	}
	if tabAllConstraints.Bio != "" {
		query = query.Where("bio = ?", tabAllConstraints.Bio)
	}
	if !tabAllConstraints.CreatedAt.IsZero() {
		query = query.Where("created_at = ?", tabAllConstraints.CreatedAt)
		// query = query.Where("DATE(created_at) = ?", tabAllConstraints.$column.goField.Format("2006-01-02"))
	}
	if !tabAllConstraints.UpdatedAt.IsZero() {
		query = query.Where("updated_at = ?", tabAllConstraints.UpdatedAt)
		// query = query.Where("DATE(updated_at) = ?", tabAllConstraints.$column.goField.Format("2006-01-02"))
	}
	if tabAllConstraints.Tinyintc != 0 {
		query = query.Where("tinyintc = ?", tabAllConstraints.Tinyintc)
	}
	if tabAllConstraints.Smallints != 0 {
		query = query.Where("smallints = ?", tabAllConstraints.Smallints)
	}
	if tabAllConstraints.Mediumint != 0 {
		query = query.Where("mediumint = ?", tabAllConstraints.Mediumint)
	}
	if tabAllConstraints.Bigintc != 0 {
		query = query.Where("bigintc = ?", tabAllConstraints.Bigintc)
	}
	if tabAllConstraints.Tinyintcu != 0 {
		query = query.Where("tinyintcu = ?", tabAllConstraints.Tinyintcu)
	}
	if tabAllConstraints.Smallintsu != 0 {
		query = query.Where("smallintsu = ?", tabAllConstraints.Smallintsu)
	}
	if tabAllConstraints.Mediumintu != 0 {
		query = query.Where("mediumintu = ?", tabAllConstraints.Mediumintu)
	}
	if tabAllConstraints.Bigintcu != 0 {
		query = query.Where("bigintcu = ?", tabAllConstraints.Bigintcu)
	}

	if !startTime.IsZero() {
		query = query.Where("create_time >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("create_time <= ?", endTime)
	}

	// // 添加分页逻辑
	// if tabAllConstraints.PageNum > 0 && tabAllConstraints.PageSize > 0 {
	//     offset := (tabAllConstraints.PageNum - 1) * tabAllConstraints.PageSize
	//     query = query.Offset(offset).Limit(tabAllConstraints.PageSize)
	// }

	err := query.Find(&tabAllConstraintss).Error
	return tabAllConstraintss, err
}

// FindTabAllConstraintsPageList 分页查询所有约束情况示例 列表
func (tacr *TabAllConstraintsRepository) FindTabAllConstraintsPageList(tabAllConstraints model.TabAllConstraints, startTime time.Time, endTime time.Time, pageNum int, pageSize int) ([]*model.TabAllConstraints, int64, error) {
	slog.Info("TabAllConstraintsRepository.FindTabAllConstraintsPageList：", slog.Any("tabAllConstraints", tabAllConstraints))

	var (
		tabAllConstraintss []*model.TabAllConstraints
		total              int64
	)

	query := tacr.db.Table((&model.TabAllConstraints{}).TableName()).Model(&model.TabAllConstraints{})

	// 构造查询条件
	if tabAllConstraints.Id != 0 {
		query = query.Where("id = ?", tabAllConstraints.Id)
	}
	if tabAllConstraints.Username != "" {
		query = query.Where("username LIKE ?", "%"+tabAllConstraints.Username+"%")
	}
	if tabAllConstraints.Email != "" {
		query = query.Where("email = ?", tabAllConstraints.Email)
	}
	if tabAllConstraints.Phone != "" {
		query = query.Where("phone = ?", tabAllConstraints.Phone)
	}
	if tabAllConstraints.CountryCode != "" {
		query = query.Where("country_code = ?", tabAllConstraints.CountryCode)
	}
	if tabAllConstraints.Status != 0 {
		query = query.Where("status = ?", tabAllConstraints.Status)
	}
	if tabAllConstraints.Role != "" {
		query = query.Where("role = ?", tabAllConstraints.Role)
	}
	if tabAllConstraints.Age != 0 {
		query = query.Where("age = ?", tabAllConstraints.Age)
	}
	if tabAllConstraints.Balance != 0 {
		query = query.Where("balance = ?", tabAllConstraints.Balance)
	}
	if tabAllConstraints.Bio != "" {
		query = query.Where("bio = ?", tabAllConstraints.Bio)
	}
	if !tabAllConstraints.CreatedAt.IsZero() {
		query = query.Where("created_at = ?", tabAllConstraints.CreatedAt)
		// query = query.Where("DATE(created_at) = ?", tabAllConstraints.$column.goField.Format("2006-01-02"))
	}
	if !tabAllConstraints.UpdatedAt.IsZero() {
		query = query.Where("updated_at = ?", tabAllConstraints.UpdatedAt)
		// query = query.Where("DATE(updated_at) = ?", tabAllConstraints.$column.goField.Format("2006-01-02"))
	}
	if tabAllConstraints.Tinyintc != 0 {
		query = query.Where("tinyintc = ?", tabAllConstraints.Tinyintc)
	}
	if tabAllConstraints.Smallints != 0 {
		query = query.Where("smallints = ?", tabAllConstraints.Smallints)
	}
	if tabAllConstraints.Mediumint != 0 {
		query = query.Where("mediumint = ?", tabAllConstraints.Mediumint)
	}
	if tabAllConstraints.Bigintc != 0 {
		query = query.Where("bigintc = ?", tabAllConstraints.Bigintc)
	}
	if tabAllConstraints.Tinyintcu != 0 {
		query = query.Where("tinyintcu = ?", tabAllConstraints.Tinyintcu)
	}
	if tabAllConstraints.Smallintsu != 0 {
		query = query.Where("smallintsu = ?", tabAllConstraints.Smallintsu)
	}
	if tabAllConstraints.Mediumintu != 0 {
		query = query.Where("mediumintu = ?", tabAllConstraints.Mediumintu)
	}
	if tabAllConstraints.Bigintcu != 0 {
		query = query.Where("bigintcu = ?", tabAllConstraints.Bigintcu)
	}

	if !startTime.IsZero() {
		query = query.Where("create_time >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("create_time <= ?", endTime)
	}

	// 分页参数默认值
	if pageNum <= 0 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 分页数据
	// todo update create_time
	err := query.
		Count(&total).Order("create_time desc").
		Limit(pageSize).Offset((pageNum - 1) * pageSize).
		Order("create_time desc").
		Find(&tabAllConstraintss).Error

	if err != nil {
		return nil, 0, err
	}

	return tabAllConstraintss, total, nil
}
