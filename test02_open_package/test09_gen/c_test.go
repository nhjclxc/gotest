package main

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strconv"
	"test09_gen/entity/model"
	"test09_gen/entity/req"
	"test09_gen/entity/resp"
	"test09_gen/repository"
	"testing"
	"time"
)

func Test1(t *testing.T) {
	uce := model.UploadClientsEvery{
		Id:          1,
		Hostname:    "2",
		UniqueId:    "3",
		UploadSpeed: 1,
		Ctime:       time.Time{},
	}

	uce.Ctime = time.Now()

	fmt.Printf("uce = %#v \n\n\n", uce)

}

func Test2(t *testing.T) {

	uce := model.UploadClientsEvery{
		Id:          1,
		Hostname:    "2",
		UniqueId:    "3",
		UploadSpeed: 1,
		Ctime:       time.Time{},
	}

	uce.Ctime = time.Now()

	reqUCE := req.UploadClientsEveryReq{}
	reqUCE.ModelToReq(&uce)
	fmt.Printf("reqUCE = %#v \n\n\n", reqUCE)

	toModel, _ := reqUCE.ReqToModel()
	fmt.Printf("toModel = %#v \n\n\n", toModel)

}

func Test3(t *testing.T) {

	uce := model.UploadClientsEvery{
		Id:          1,
		Hostname:    "2",
		UniqueId:    "3",
		UploadSpeed: 1,
		Ctime:       time.Time{},
	}

	uce.Ctime = time.Now()

	r := resp.UploadClientsEveryResp{}
	r.ModelToResp(&uce)

	fmt.Printf("r = %#v \n\n\n", r)

}

func Test5(t *testing.T) {

	// jdbc:mysql://117.72.184.33:3306/test1?useSSL=false&serverTimezone=GMT%2B8&characterEncoding=utf8&zeroDateTimeBehavior=convertToNull
	user := "root"
	password := "root123"
	host := "117.72.184.33"
	port := 3306
	dbname := "test1"
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, dbname)

	fmt.Println(dsn)
	logLevel := logger.Info
	db, dberr := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if dberr != nil {
		fmt.Println(" dberr ", dberr)
		return
	}

	repo := repository.NewUploadClientsEveryRepository(db)

	uce1 := model.UploadClientsEvery{
		Hostname: "qqq11ascwwwwa111",
		Ctime:    time.Now(),
	}
	uce2 := model.UploadClientsEvery{
		Hostname:    "qqqq222wwqwq22",
		UniqueId:    "2wq22222",
		UploadSpeed: 2,
	}
	datasptr := []*model.UploadClientsEvery{&uce1, &uce2}
	datas := []model.UploadClientsEvery{uce1, uce2}
	// 值类型切片
	result := db.Create(&datas)
	fmt.Println("RowsAffected:", result.RowsAffected, "Err:", result.Error)

	// 指针类型切片
	result2 := db.Create(&datasptr)
	fmt.Println("RowsAffected:", result2.RowsAffected, "Err:", result2.Error)
	_ = datasptr
	_ = datas

	if true {
		return
	}
	//
	_ = uce1
	_ = uce2

	uce := model.UploadClientsEvery{
		UniqueId: "22",
	}
	var satrtTime time.Time
	var endTime time.Time
	list, i, err := repo.FindUploadClientsEveryPageList(uce, satrtTime, endTime, 2, 2)
	fmt.Println(i)
	fmt.Println(list)
	fmt.Println(" err ", err)

	//
	//if true {
	//	return
	//}
	//clientsEvery, err := repo.BatchDeleteUploadClientsEvery([]int64{3, 4})
	//fmt.Println("clientsEvery ", clientsEvery, " err ", err)

}

type TestRec struct {
	num int
}

func (t TestRec) func2() int {
	t.num = t.num * 2
	return t.num
}
func (t *TestRec) func3() int {
	t.num = t.num * 3
	return t.num
}

func Test6(t *testing.T) {

	t1 := TestRec{num: 1}
	fmt.Println("t1.func2 ", t1.func2(), "num", t1.num)
	fmt.Println("t1.func3 ", t1.func3(), "num", t1.num)
	fmt.Println("(&t1).func2 ", (&t1).func2(), "num", t1.num)
	fmt.Println("(&t1).func3 ", (&t1).func3(), "num", t1.num)

	t2 := &TestRec{num: 2}
	fmt.Println("(*t2).func2 ", (*t2).func2(), "num", t2.num)
	fmt.Println("(*t2).func3 ", (*t2).func3(), "num", t2.num)
	fmt.Println("t2.func3 ", t2.func2(), "num", t2.num)
	fmt.Println("t2.func3 ", t2.func3(), "num", t2.num)

}

//
//upload_clients_every表对应的repository.go如下，你认为每一个方法以及方法签名是否正确？？？
//```upload_clients_every_repository.go
//type UploadClientsEveryRepository struct {
//	db *gorm.DB
//}
//
//// NewUploadClientsEveryRepository 创建UploadClientsEvery 结构体持久层对象
//func NewUploadClientsEveryRepository(db *gorm.DB) *UploadClientsEveryRepository {
//	return &UploadClientsEveryRepository{
//		db: db,
//	}
//}
//
//// InsertUploadClientsEvery 新增UploadClientsEvery
//func (ucer *UploadClientsEveryRepository) InsertUploadClientsEvery(uploadClientsEvery model.UploadClientsEvery) (int, error)
//
//// BatchInsertUploadClientsEverys 批量新增UploadClientsEvery
//func (ucer *UploadClientsEveryRepository) BatchInsertUploadClientsEverys(uploadClientsEverys []*model.UploadClientsEvery) (int, error)
//
//// UpdateUploadClientsEveryById 根据主键修改UploadClientsEvery的所有字段
//func (ucer *UploadClientsEveryRepository) UpdateUploadClientsEveryById(uploadClientsEvery model.UploadClientsEvery) (int, error)
//
//// UpdateUploadClientsEverySelective 修改UploadClientsEvery不为默认值的字段
//func (ucer *UploadClientsEveryRepository) UpdateUploadClientsEverySelective(uploadClientsEvery model.UploadClientsEvery) (int, error)
//
//// BatchUpdateUploadClientsEverySelective 批量修改UploadClientsEvery
//func (ucer *UploadClientsEveryRepository) BatchUpdateUploadClientsEverySelective(uploadClientsEverys []model.UploadClientsEvery) error
//
//// BatchDeleteUploadClientsEveryByState 批量软删除 UploadClientsEvery
//func (ucer *UploadClientsEveryRepository) BatchDeleteUploadClientsEveryByState(idList []int64) error
//
//// BatchDeleteUploadClientsEvery 根据主键批量删除 UploadClientsEvery
//func (ucer *UploadClientsEveryRepository) BatchDeleteUploadClientsEvery(idList []int64) (int, error)
//
//// FindUploadClientsEveryById 获取UploadClientsEvery详细信息
//func (ucer *UploadClientsEveryRepository) FindUploadClientsEveryById(id int64) (model.UploadClientsEvery, error)
//
//// FindUploadClientsEverysByIdList 根据主键批量查询UploadClientsEvery详细信息
//func (ucer *UploadClientsEveryRepository) FindUploadClientsEverysByIdList(idList []int64) ([]model.UploadClientsEvery, error)
//
//// FindUploadClientsEveryList 查询UploadClientsEvery列表
//func (ucer *UploadClientsEveryRepository) FindUploadClientsEveryList(uploadClientsEvery model.UploadClientsEvery, satrtTime time.Time, endTime time.Time) ([]model.UploadClientsEvery, error)
//
//// FindUploadClientsEveryPageList 分页查询UploadClientsEvery列表
//func (ucer *UploadClientsEveryRepository) FindUploadClientsEveryPageList(uploadClientsEvery model.UploadClientsEvery, satrtTime time.Time, endTime time.Time, pageNum int, pageSize int) ([]model.UploadClientsEvery, int64, error)
//
//```

// TabUser 用户 结构体
// @author
// @date 2025-08-25T15:25:53.069819
type TabUser struct {
	Id int64 `gorm:"column:id;type:bigint;primaryKey;autoIncrement;not null;comment:用户ID" json:"id" form:"id"` // 用户ID

	Username string `gorm:"column:username;type:varchar(50);unique;not null;index:idx_username_email,priority:1;comment:用户名" json:"username" form:"username"` // 用户名

	Password string `gorm:"column:password;type:varchar(255);not null;comment:密码（加密存储）" json:"password" form:"password"` // 密码（加密存储）

	Email string `gorm:"column:email;type:varchar(100);index:idx_username_email,priority:2;comment:邮箱" json:"email" form:"email"` // 邮箱

	CreatedAt time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:创建时间" json:"createdAt" form:"createdAt"` // 创建时间

	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间" json:"updatedAt" form:"updatedAt"` // 更新时间

}

// TableName 返回当前实体类的表名
func (tu *TabUser) TableName() string {
	return "tab_user" + strconv.FormatInt(time.Now().Unix(), 10)
}

// CreateTable 根据结构体里面的gorm信息创建表结构
func (tu *TabUser) CreateTable(tx *gorm.DB) error {
	tableName := tu.TableName()
	if !tx.Migrator().HasTable(tableName) {
		err := tx.Table(tableName).Migrator().CreateTable(&TabUser{})
		if err != nil {
			return err
		}
	}
	return nil
}

func Test123(t *testing.T) {

	db, done := getDB()
	if done {
		return
	}

	u := model.UploadClientsEvery{}

	u.CreateTable(db)

}

func getDB() (*gorm.DB, bool) {
	// jdbc:mysql://117.72.184.33:3306/test1?useSSL=false&serverTimezone=GMT%2B8&characterEncoding=utf8&zeroDateTimeBehavior=convertToNull
	user := "root"
	password := "root123"
	host := "117.72.184.33"
	port := 3306
	dbname := "test1"
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, dbname)

	fmt.Println(dsn)
	logLevel := logger.Info
	db, dberr := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if dberr != nil {
		fmt.Println(" dberr ", dberr)
		return nil, true
	}
	return db, false
}

// TabAllConstraints 所有约束情况示例 结构体
// @author
// @date 2025-08-25T15:39:58.175933
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
	return "tab_all_constraints111" + strconv.FormatInt(time.Now().Unix(), 10)
}

// CreateTable 根据结构体里面的gorm信息创建表结构
func (tac *TabAllConstraints) CreateTable(tx *gorm.DB) error {
	tableName := tac.TableName()
	if !tx.Migrator().HasTable(tableName) {
		err := tx.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COMMENT='用户222表'").
			Table(tableName).Migrator().CreateTable(&TabAllConstraints{})
		if err != nil {
			return err
		}
	}
	return nil
}

func Test23(t *testing.T) {

	db, done := getDB()
	if done {
		return
	}
	a := TabAllConstraints{}
	a.CreateTable(db)
}

func Test233(t *testing.T) {

	db, done := getDB()
	if done {
		return
	}

	u := model.UploadClientsEvery{}

	u.CreateTable(db)

}
