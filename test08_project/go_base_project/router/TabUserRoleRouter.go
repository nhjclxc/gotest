package router

import (
	"github.com/gin-gonic/gin"
)

// TabUserRoleRouter 用户角色关联 路由Router层
type TabUserRoleRouter struct {
	tabUserRoleApi api.TabUserRoleApi
}

// InitTabUserRoleRouter 初始化 TabUserRoleRouter 路由
func (this *TabUserRoleRouter) InitTabUserRoleRouter(privateRouterOrigin *gin.RouterGroup, publicRouterOrigin *gin.RouterGroup) {
	privateRouter := privateRouterOrigin.Group("/tab/user/role")
	{
		// PrivateRouter 下是一些必须进行登录的接口
		// http://localhost:8080/private

		privateRouter.POST("", this.tabUserRoleApi.InsertTabUserRole)              // 新增用户角色关联
		privateRouter.PUT("", this.tabUserRoleApi.UpdateTabUserRole)               // 修改用户角色关联
		privateRouter.DELETE("/:idList", this.tabUserRoleApi.DeleteTabUserRole)    // 删除用户角色关联
		privateRouter.GET("/:id", this.tabUserRoleApi.GetTabUserRoleById)          // 获取用户角色关联详细信息
		privateRouter.GET("/list", this.tabUserRoleApi.GetTabUserRoleList)         // 查询用户角色关联列表
		privateRouter.GET("/pageList", this.tabUserRoleApi.GetTabUserRolePageList) // 分页查询用户角色关联列表
		privateRouter.GET("/export", this.tabUserRoleApi.ExportTabUserRole)        // 导出用户角色关联列表
	}

	publicRouter := publicRouterOrigin.Group("/tab/user/role")
	{
		// PublicRouter 下是一些无需登录的接口，可以直接访问，无须经过授权操作
		// http://localhost:8080/public

		publicRouter.POST("", this.tabUserRoleApi.InsertTabUserRole)              // 新增用户角色关联
		publicRouter.PUT("", this.tabUserRoleApi.UpdateTabUserRole)               // 修改用户角色关联
		publicRouter.DELETE("/:id", this.tabUserRoleApi.DeleteTabUserRole)        // 删除用户角色关联
		publicRouter.GET("/:id", this.tabUserRoleApi.GetTabUserRoleById)          // 获取用户角色关联详细信息
		publicRouter.GET("/list", this.tabUserRoleApi.GetTabUserRoleList)         // 查询用户角色关联列表
		publicRouter.GET("/pageList", this.tabUserRoleApi.GetTabUserRolePageList) // 分页查询用户角色关联列表
		publicRouter.GET("/export", this.tabUserRoleApi.ExportTabUserRole)        // 导出用户角色关联列表
	}
}
