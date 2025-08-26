package router

import (
	"github.com/gin-gonic/gin"
	"test09_gen/controller"
)

// UploadClientsEveryRouter UploadClientsEvery 路由Router层
type UploadClientsEveryRouter struct {
	uploadClientsEveryController *controller.UploadClientsEveryController
}

// NewUploadClientsEveryRouter 创建 UploadClientsE2very UploadClientsEvery 路由Router层
func NewUploadClientsEveryRouter(uploadClientsEveryController *controller.UploadClientsEveryController) *UploadClientsEveryRouter {
	return &UploadClientsEveryRouter{
		uploadClientsEveryController: uploadClientsEveryController,
	}
}

// InitUploadClientsEveryRouter 初始化 UploadClientsEveryRouter 路由
func (ucer *UploadClientsEveryRouter) InitUploadClientsEveryRouter(privateRouterOrigin *gin.RouterGroup, publicRouterOrigin *gin.RouterGroup) {
	privateRouter := privateRouterOrigin.Group("/upload/clients/every")
	{
		// PrivateRouter 下是一些必须进行登录的接口
		// http://localhost:8080/private

		privateRouter.POST("", ucer.uploadClientsEveryController.InsertUploadClientsEvery)              // 新增UploadClientsEvery
		privateRouter.PUT("", ucer.uploadClientsEveryController.UpdateUploadClientsEvery)               // 修改UploadClientsEvery
		privateRouter.DELETE("/:idList", ucer.uploadClientsEveryController.DeleteUploadClientsEvery)    // 删除UploadClientsEvery
		privateRouter.GET("/:id", ucer.uploadClientsEveryController.GetUploadClientsEveryById)          // 获取UploadClientsEvery详细信息
		privateRouter.GET("/list", ucer.uploadClientsEveryController.GetUploadClientsEveryList)         // 查询UploadClientsEvery列表
		privateRouter.GET("/pageList", ucer.uploadClientsEveryController.GetUploadClientsEveryPageList) // 分页查询UploadClientsEvery列表
		privateRouter.GET("/export", ucer.uploadClientsEveryController.ExportUploadClientsEvery)        // 导出UploadClientsEvery列表
	}

	publicRouter := publicRouterOrigin.Group("/upload/clients/every")
	{
		// PublicRouter 下是一些无需登录的接口，可以直接访问，无须经过授权操作
		// http://localhost:8080/public

		publicRouter.POST("", ucer.uploadClientsEveryController.InsertUploadClientsEvery)              // 新增UploadClientsEvery
		publicRouter.PUT("", ucer.uploadClientsEveryController.UpdateUploadClientsEvery)               // 修改UploadClientsEvery
		publicRouter.DELETE("/:id", ucer.uploadClientsEveryController.DeleteUploadClientsEvery)        // 删除UploadClientsEvery
		publicRouter.GET("/:id", ucer.uploadClientsEveryController.GetUploadClientsEveryById)          // 获取UploadClientsEvery详细信息
		publicRouter.GET("/list", ucer.uploadClientsEveryController.GetUploadClientsEveryList)         // 查询UploadClientsEvery列表
		publicRouter.GET("/pageList", ucer.uploadClientsEveryController.GetUploadClientsEveryPageList) // 分页查询UploadClientsEvery列表
		publicRouter.GET("/export", ucer.uploadClientsEveryController.ExportUploadClientsEvery)        // 导出UploadClientsEvery列表
	}
}
