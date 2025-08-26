package model

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"gorm.io/gorm"
	"reflect"
	"strings"
	"time"
)

// TabAllConstraints 所有约束情况示例 结构体
// @author
// @date 2025-08-25T16:18:49.753
type TabAllConstraints struct {
	Id uint64 `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement;not null;comment:主键ID" json:"id" form:"id"` // 主键ID

	Username string `gorm:"column:username;type:varchar(50);unique;not null;comment:用户名（唯一）" json:"username" form:"username"` // 用户名（唯一）

	Email string `gorm:"column:email;type:varchar(100);index:idx_email,priority:1;comment:邮箱" json:"email" form:"email"` // 邮箱

	Phone string `gorm:"column:phone;type:varchar(20);index:uniq_phone_country,priority:1;comment:手机号" json:"phone" form:"phone"` // 手机号

	CountryCode string `gorm:"column:country_code;type:char(5);default:+86;index:uniq_phone_country,priority:2;comment:国家区号" json:"countryCode" form:"countryCode"` // 国家区号

	Status int8 `gorm:"column:status;type:tinyint;not null;default:1;comment:状态：1=正常，0=禁用" json:"status" form:"status"` // 状态：1=正常，0=禁用

	Role string `gorm:"column:role;type:enum('admin','user','guest');not null;default:user;comment:角色" json:"role" form:"role"` // 角色

	Age int32 `gorm:"column:age;type:int;default:0;comment:年龄" json:"age" form:"age"` // 年龄

	Balance float64 `gorm:"column:balance;type:decimal(10,2);not null;default:0.00;comment:账户余额" json:"balance" form:"balance"` // 账户余额

	Bio string `gorm:"column:bio;type:text;comment:用户简介" json:"bio" form:"bio"` // 用户简介

	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:创建时间" json:"createdAt" form:"createdAt"` // 创建时间

	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间" json:"updatedAt" form:"updatedAt"` // 更新时间

	Tinyintc int8 `gorm:"column:tinyintc;type:tinyint" json:"tinyintc" form:"tinyintc"` //

	Smallints int16 `gorm:"column:smallints;type:smallint" json:"smallints" form:"smallints"` //

	Mediumint int32 `gorm:"column:mediumint;type:mediumint" json:"mediumint" form:"mediumint"` //

	Bigintc int64 `gorm:"column:bigintc;type:bigint" json:"bigintc" form:"bigintc"` //

	Tinyintcu uint8 `gorm:"column:tinyintcu;type:tinyint unsigned" json:"tinyintcu" form:"tinyintcu"` //

	Smallintsu uint16 `gorm:"column:smallintsu;type:smallint unsigned" json:"smallintsu" form:"smallintsu"` //

	Mediumintu uint32 `gorm:"column:mediumintu;type:mediumint unsigned" json:"mediumintu" form:"mediumintu"` //

	Bigintcu uint64 `gorm:"column:bigintcu;type:bigint unsigned" json:"bigintcu" form:"bigintcu"` //

}

// TableName 返回当前实体类的表名
func (tac *TabAllConstraints) TableName() string {
	return "tab_all_constraints"
}

// CreateTable 根据结构体里面的gorm信息创建表结构
func (tac *TabAllConstraints) CreateTable(tx *gorm.DB) error {
	tableName := tac.TableName()
	if !tx.Migrator().HasTable(tableName) {
		err := tx.Table(tableName).Migrator().CreateTable(&TabAllConstraints{})
		if err != nil {
			return err
		}
	}
	return nil
}

// 可用钩子函数包括：BeforeCreate / AfterCreate、BeforeUpdate / AfterUpdate、BeforeDelete / AfterDelete
// BeforeCreate 在插入数据之前执行的操作
func (tac *TabAllConstraints) BeforeCreate(tx *gorm.DB) (err error) {
	if err = tac.CreateTable(tx); err != nil {
		return err
	}

	return
}

func (tac *TabAllConstraints) BeforeUpdate(tx *gorm.DB) (err error) {
	return
}

// MapToTabAllConstraints map映射转化为当前结构体
func MapToTabAllConstraints(inputMap map[string]any) *TabAllConstraints {
	//go get github.com/mitchellh/mapstructure

	var tabAllConstraints TabAllConstraints
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
				if from.Kind() == reflect.String && to == reflect.TypeOf(time.Time{}) {
					return time.Parse("2006-01-02 15:04:05", data.(string))
				}
				return data, nil
			},
		),
		Result: &tabAllConstraints,
	})
	if err != nil {
		fmt.Printf("MapToStruct Decode NewDecoder error: %v", err)
		panic(err)
	}

	if err := decoder.Decode(inputMap); err != nil {
		fmt.Printf("MapToStruct Decode error: %v", err)
	}
	return &tabAllConstraints
}

// TabAllConstraintsToMap 当前结构体转化为map映射
func (tac *TabAllConstraints) TabAllConstraintsToMap() map[string]any {
	// 先转成 map
	m := make(map[string]any)
	bytes, err := json.Marshal(tac)
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
