package model

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"gorm.io/gorm"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// UploadClientsEvery UploadClientsEvery结构体
// @author
// @date 2025-08-25T15:32:51.053028
type UploadClientsEvery struct {
	Id int `gorm:"column:id;type:int unsigned;primaryKey;autoIncrement;not null" json:"id" form:"id"` //

	Hostname string `gorm:"column:hostname;type:varchar(255);index:idx_hostname_unique_id,priority:1;index:upload_clients_hostname_unique_id_idx,priority:1" json:"hostname" form:"hostname"` //

	UniqueId string `gorm:"column:unique_id;type:varchar(255);index:idx_hostname_unique_id,priority:2;index:upload_clients_hostname_unique_id_idx,priority:2" json:"uniqueId" form:"uniqueId"` //

	UploadSpeed int `gorm:"column:upload_speed;type:double;comment:bytes/s" json:"uploadSpeed" form:"uploadSpeed"` // bytes/s

	Ctime time.Time `gorm:"column:ctime;type:datetime;index:upload_clients_hostname_unique_id_idx,priority:3" json:"ctime" form:"ctime"` //

}

// TableName 返回当前实体类的表名
func (uce *UploadClientsEvery) TableName() string {
	return "upload_clients_every" + strconv.FormatInt(time.Now().Unix(), 10)
}

// CreateTable 根据结构体里面的gorm信息创建表结构
func (uce *UploadClientsEvery) CreateTable(tx *gorm.DB) error {
	tableName := uce.TableName()
	if !tx.Migrator().HasTable(tableName) {
		err := tx.Table(tableName).Migrator().CreateTable(&UploadClientsEvery{})
		if err != nil {
			return err
		}
	}
	return nil
}

// 可用钩子函数包括：BeforeCreate / AfterCreate、BeforeUpdate / AfterUpdate、BeforeDelete / AfterDelete
// BeforeCreate 在插入数据之前执行的操作
func (uce *UploadClientsEvery) BeforeCreate(tx *gorm.DB) (err error) {
	if err = uce.CreateTable(tx); err != nil {
		return err
	}

	return
}

func (uce *UploadClientsEvery) BeforeUpdate(tx *gorm.DB) (err error) {
	return
}

// MapToUploadClientsEvery map映射转化为当前结构体
func MapToUploadClientsEvery(inputMap map[string]any) *UploadClientsEvery {
	//go get github.com/mitchellh/mapstructure

	var uploadClientsEvery UploadClientsEvery
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
				if from.Kind() == reflect.String && to == reflect.TypeOf(time.Time{}) {
					return time.Parse("2006-01-02 15:04:05", data.(string))
				}
				return data, nil
			},
		),
		Result: &uploadClientsEvery,
	})
	if err != nil {
		fmt.Printf("MapToStruct Decode NewDecoder error: %v", err)
		panic(err)
	}

	if err := decoder.Decode(inputMap); err != nil {
		fmt.Printf("MapToStruct Decode error: %v", err)
	}
	return &uploadClientsEvery
}

// UploadClientsEveryToMap 当前结构体转化为map映射
func (uce *UploadClientsEvery) UploadClientsEveryToMap() map[string]any {
	// 先转成 map
	m := make(map[string]any)
	bytes, err := json.Marshal(uce)
	if err != nil {
		fmt.Printf("StructToMap marshal error: %v", err)
		return nil
	}

	err = json.Unmarshal(bytes, &m)
	if err != nil {
		fmt.Printf("StructToMap unmarshal error: %v", err)
		return nil
	}

	// 格式化所有 time.Time
	for k, v := range m {
		if t, ok := v.(string); ok && len(t) > 10 && strings.Contains(t, "T") {
			// 尝试解析 RFC3339，再格式化
			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				m[k] = parsed.Format("2006-01-02 15:04:05")
			}
		}
	}

	return m
}

// 东八区
var cstZone = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}()

// 获取东八区时间
func getNow() time.Time {
	return time.Now().In(cstZone)
}
