package router

import (
	"github.com/gin-gonic/gin"
)

// TabRoleRouter 角色 路由Router层
type TabRoleRouter struct {
	tabRoleApi api.TabRoleApi
}

// InitTabRoleRouter 初始化 TabRoleRouter 路由
func (this *TabRoleRouter) InitTabRoleRouter(privateRouterOrigin *gin.RouterGroup, publicRouterOrigin *gin.RouterGroup) {
	privateRouter := privateRouterOrigin.Group("/tab/role")
	{
		// PrivateRouter 下是一些必须进行登录的接口
		// http://localhost:8080/private

		privateRouter.POST("", this.tabRoleApi.InsertTabRole)              // 新增角色
		privateRouter.PUT("", this.tabRoleApi.UpdateTabRole)               // 修改角色
		privateRouter.DELETE("/:idList", this.tabRoleApi.DeleteTabRole)    // 删除角色
		privateRouter.GET("/:id", this.tabRoleApi.GetTabRoleById)          // 获取角色详细信息
		privateRouter.GET("/list", this.tabRoleApi.GetTabRoleList)         // 查询角色列表
		privateRouter.GET("/pageList", this.tabRoleApi.GetTabRolePageList) // 分页查询角色列表
		privateRouter.GET("/export", this.tabRoleApi.ExportTabRole)        // 导出角色列表
	}

	publicRouter := publicRouterOrigin.Group("/tab/role")
	{
		// PublicRouter 下是一些无需登录的接口，可以直接访问，无须经过授权操作
		// http://localhost:8080/public

		publicRouter.POST("", this.tabRoleApi.InsertTabRole)              // 新增角色
		publicRouter.PUT("", this.tabRoleApi.UpdateTabRole)               // 修改角色
		publicRouter.DELETE("/:id", this.tabRoleApi.DeleteTabRole)        // 删除角色
		publicRouter.GET("/:id", this.tabRoleApi.GetTabRoleById)          // 获取角色详细信息
		publicRouter.GET("/list", this.tabRoleApi.GetTabRoleList)         // 查询角色列表
		publicRouter.GET("/pageList", this.tabRoleApi.GetTabRolePageList) // 分页查询角色列表
		publicRouter.GET("/export", this.tabRoleApi.ExportTabRole)        // 导出角色列表
	}
}
