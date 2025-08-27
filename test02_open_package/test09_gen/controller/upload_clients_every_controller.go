package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"test09_gen/entity/model"
	"test09_gen/entity/req"
	"test09_gen/service"
	commonUtils "test09_gen/utils"
)

// UploadClientsEveryController UploadClientsEvery 控制器层
type UploadClientsEveryController struct {
	uploadClientsEveryService *service.UploadClientsEveryService
}

// NewUploadClientsEveryController 创建 UploadClientsEveloadClientsEvery 控制器层对象
func NewUploadClientsEveryController(uploadClientsEveryService *service.UploadClientsEveryService) *UploadClientsEveryController {
	return &UploadClientsEveryController{
		uploadClientsEveryService: uploadClientsEveryService,
	}
}

// InsertUploadClientsEvery 新增UploadClientsEvery
// @Tags UploadClientsEvery模块
// @Summary 新增UploadClientsEvery-Summary
// @Description 新增UploadClientsEvery-Description
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uploadClientsEvery body model.UploadClientsEvery true "修改UploadClientsEvery实体类"
// @Success 200 {object} commonUtils.JsonResult "新增UploadClientsEvery响应数据"
// @Failure 401 {object} commonUtils.JsonResult "未授权"
// @Failure 500 {object} commonUtils.JsonResult "服务器异常"
// @Router /upload/clients/every [post]
func (ucec *UploadClientsEveryController) InsertUploadClientsEvery(c *gin.Context) {
	var uploadClientsEvery model.UploadClientsEvery
	c.ShouldBindJSON(&uploadClientsEvery)

	res, err := ucec.uploadClientsEveryService.InsertUploadClientsEvery(uploadClientsEvery)

	if err != nil {
		c.JSON(http.StatusInternalServerError, commonUtils.JsonResultError("新增UploadClientsEvery失败："+err.Error()))
		return
	}
	c.JSON(http.StatusOK, commonUtils.JsonResultSuccess[any](res))
}

// UpdateUploadClientsEvery 修改UploadClientsEvery
// @Tags UploadClientsEvery模块
// @Summary 修改UploadClientsEvery-Summary
// @Description 修改UploadClientsEvery-Description
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param uploadClientsEvery body model.UploadClientsEvery true "修改UploadClientsEvery实体类"
// @Success 200 {object} commonUtils.JsonResult "修改UploadClientsEvery响应数据"
// @Failure 401 {object} commonUtils.JsonResult "未授权"
// @Failure 500 {object} commonUtils.JsonResult "服务器异常"
// @Router /upload/clients/every [put]
func (ucec *UploadClientsEveryController) UpdateUploadClientsEvery(c *gin.Context) {
	var uploadClientsEvery model.UploadClientsEvery
	c.ShouldBindJSON(&uploadClientsEvery)

	res, err := ucec.uploadClientsEveryService.UpdateUploadClientsEvery(uploadClientsEvery)

	if err != nil {
		c.JSON(http.StatusInternalServerError, commonUtils.JsonResultError("修改UploadClientsEvery失败："+err.Error()))
		return
	}
	c.JSON(http.StatusOK, commonUtils.JsonResultSuccess[any](res))
}

// DeleteUploadClientsEvery 删除UploadClientsEvery
// @Tags UploadClientsEvery模块
// @Summary 删除UploadClientsEvery-Summary
// @Description 删除UploadClientsEvery-Description
// @Security BearerAuth
// @Accept path
// @Produce path
// @Param idList path string true "UploadClientsEvery主键List"
// @Success 200 {object} commonUtils.JsonResult "删除UploadClientsEvery响应数据"
// @Failure 401 {object} commonUtils.JsonResult "未授权"
// @Failure 500 {object} commonUtils.JsonResult "服务器异常"
// @Router /upload/clients/every/:idList [delete]
func (ucec *UploadClientsEveryController) DeleteUploadClientsEvery(c *gin.Context) {
	idListStr := c.Param("idList") // 例如: "1,2,3"
	idList, err := commonUtils.ParseIds(idListStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, commonUtils.JsonResultError("参数错误："+err.Error()))
		return
	}

	res, err := ucec.uploadClientsEveryService.DeleteUploadClientsEvery(idList)

	if err != nil {
		c.JSON(http.StatusInternalServerError, commonUtils.JsonResultError("删除UploadClientsEvery失败："+err.Error()))
		return
	}
	c.JSON(http.StatusOK, commonUtils.JsonResultSuccess[any](res))
}

// GetUploadClientsEveryById 获取UploadClientsEvery详细信息
// @Tags UploadClientsEvery模块
// @Summary 获取UploadClientsEvery详细信息-Summary
// @Description 获取UploadClientsEvery详细信息-Description
// @Security BearerAuth
// @Accept path
// @Produce path
// @Param id path int64 true "UploadClientsEvery主键List"
// @Success 200 {object} commonUtils.JsonResult "获取UploadClientsEvery详细信息"
// @Failure 401 {object} commonUtils.JsonResult "未授权"
// @Failure 500 {object} commonUtils.JsonResult "服务器异常"
// @Router /upload/clients/every/:id [get]
func (ucec *UploadClientsEveryController) GetUploadClientsEveryById(c *gin.Context) {
	idStr := c.Param("id") // 例如: "1"
	id, err := commonUtils.ParseId(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, commonUtils.JsonResultError("参数错误："+err.Error()))
		return
	}

	res, err := ucec.uploadClientsEveryService.GetUploadClientsEveryById(id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, commonUtils.JsonResultError("查询UploadClientsEvery失败："+err.Error()))
		return
	}
	c.JSON(http.StatusOK, commonUtils.JsonResultSuccess[any](res))
}

// GetUploadClientsEveryList 查询UploadClientsEvery列表
// @Tags UploadClientsEvery模块
// @Summary 查询UploadClientsEvery列表-Summary
// @Description 查询UploadClientsEvery列表-Description
// @Security BearerAuth
// @Accept param
// @Produce param
// @Param uploadClientsEveryReq body req.UploadClientsEveryReq true "UploadClientsEvery实体Req"
// @Success 200 {object} commonUtils.JsonResult "查询UploadClientsEvery列表响应数据"
// @Failure 401 {object} commonUtils.JsonResult "未授权"
// @Failure 500 {object} commonUtils.JsonResult "服务器异常"
// @Router /upload/clients/every/list [get]
func (ucec *UploadClientsEveryController) GetUploadClientsEveryList(c *gin.Context) {
	var uploadClientsEveryReq req.UploadClientsEveryReq
	c.ShouldBindQuery(&uploadClientsEveryReq)

	res, err := ucec.uploadClientsEveryService.GetUploadClientsEveryList(uploadClientsEveryReq)

	if err != nil {
		c.JSON(http.StatusInternalServerError, commonUtils.JsonResultError("查询UploadClientsEvery列表失败："+err.Error()))
		return
	}
	c.JSON(http.StatusOK, commonUtils.JsonResultSuccess[any](res))
}

// GetUploadClientsEveryPageList 分页查询UploadClientsEvery列表
// @Tags UploadClientsEvery模块
// @Summary 分页查询UploadClientsEvery列表-Summary
// @Description 分页查询UploadClientsEvery列表-Description
// @Security BearerAuth
// @Accept param
// @Produce param
// @Param uploadClientsEveryReq body req.UploadClientsEveryReq true "UploadClientsEvery实体Req"
// @Success 200 {object} commonUtils.JsonResult "分页查询UploadClientsEvery列表响应数据"
// @Failure 401 {object} commonUtils.JsonResult "未授权"
// @Failure 500 {object} commonUtils.JsonResult "服务器异常"
// @Router /upload/clients/every/pageList [get]
func (ucec *UploadClientsEveryController) GetUploadClientsEveryPageList(c *gin.Context) {
	var uploadClientsEveryReq req.UploadClientsEveryReq
	c.ShouldBindQuery(&uploadClientsEveryReq)

	res, err := ucec.uploadClientsEveryService.GetUploadClientsEveryPageList(uploadClientsEveryReq)

	if err != nil {
		c.JSON(http.StatusInternalServerError, commonUtils.JsonResultError("查询UploadClientsEvery列表失败："+err.Error()))
		return
	}
	c.JSON(http.StatusOK, commonUtils.JsonResultSuccess[any](res))
}

// ExportUploadClientsEvery 导出UploadClientsEvery列表
// @Tags UploadClientsEvery模块
// @Summary 导出UploadClientsEvery列表-Summary
// @Description 导出UploadClientsEvery列表-Description
// @Security BearerAuth
// @Accept param
// @Produce param
// @Param uploadClientsEveryReq body req.UploadClientsEveryReq true "UploadClientsEvery实体Req"
// @Success 200 {object} commonUtils.JsonResult "导出UploadClientsEvery列表响应数据"
// @Failure 401 {object} commonUtils.JsonResult "未授权"
// @Failure 500 {object} commonUtils.JsonResult "服务器异常"
// @Router /upload/clients/every/export [get]
func (ucec *UploadClientsEveryController) ExportUploadClientsEvery(c *gin.Context) {
	var uploadClientsEveryReq req.UploadClientsEveryReq
	c.ShouldBindQuery(&uploadClientsEveryReq)

	res, err := ucec.uploadClientsEveryService.ExportUploadClientsEvery(uploadClientsEveryReq)

	if err != nil {
		c.JSON(http.StatusInternalServerError, commonUtils.JsonResultError("导出UploadClientsEvery列表失败："+err.Error()))
		return
	}
	c.JSON(http.StatusOK, commonUtils.JsonResultSuccess[any](res))
}
