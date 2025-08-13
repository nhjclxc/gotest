package service

// TabUserRoleService 用户角色关联 Service 层
type TabUserRoleService struct {
}

// InsertTabUserRole 新增用户角色关联
func (this *TabUserRoleService) InsertTabUserRole(tabUserRole *model.TabUserRole) (res any, err error) {

	return tabUserRole.InsertTabUserRole(core.GLOBAL_DB)
}

// UpdateTabUserRole 修改用户角色关联
func (this *TabUserRoleService) UpdateTabUserRole(tabUserRole *model.TabUserRole) (res any, err error) {

	return tabUserRole.UpdateTabUserRoleById(core.GLOBAL_DB)
}

// DeleteTabUserRole 删除用户角色关联
func (this *TabUserRoleService) DeleteTabUserRole(idList []int64) (res any, err error) {

	return (&model.TabUserRole{}).DeleteTabUserRole(core.GLOBAL_DB, idList)
}

// GetTabUserRoleById 获取用户角色关联业务详细信息
func (this *TabUserRoleService) GetTabUserRoleById(id int64) (res any, err error) {

	tabUserRole := model.TabUserRole{}
	err = (&tabUserRole).FindTabUserRoleById(core.GLOBAL_DB, id)
	if err != nil {
		return nil, err
	}

	return tabUserRole, nil
}

// GetTabUserRoleList 查询用户角色关联业务列表
func (this *TabUserRoleService) GetTabUserRoleList(tabUserRoleDto *dto.TabUserRoleDto) (res any, err error) {

	tabUserRole, err := tabUserRoleDto.DtoToModel()
	tabUserRoleList, err := tabUserRole.FindTabUserRoleList(core.GLOBAL_DB, tabUserRoleDto.SatrtTime, tabUserRoleDto.EndTime)
	if err != nil {
		return nil, err
	}

	return tabUserRoleList, nil
}

// GetTabUserRolePageList 分页查询用户角色关联业务列表
func (this *TabUserRoleService) GetTabUserRolePageList(tabUserRoleDto *dto.TabUserRoleDto) (res any, err error) {

	tabUserRole, err := tabUserRoleDto.DtoToModel()
	tabUserRoleList, total, err := tabUserRole.FindTabUserRolePageList(core.GLOBAL_DB, tabUserRoleDto.SatrtTime, tabUserRoleDto.EndTime, tabUserRoleDto.PageNum, tabUserRoleDto.PageSize)
	if err != nil {
		return nil, err
	}

	return commonUtils.BuildPageData[model.TabUserRole](tabUserRoleList, total, tabUserRoleDto.PageNum, tabUserRoleDto.PageSize), nil
}

// ExportTabUserRole 导出用户角色关联业务列表
func (this *TabUserRoleService) ExportTabUserRole(tabUserRoleDto *dto.TabUserRoleDto) (res any, err error) {

	tabUserRole, err := tabUserRoleDto.DtoToModel()
	tabUserRole.FindTabUserRolePageList(core.GLOBAL_DB, tabUserRoleDto.SatrtTime, tabUserRoleDto.EndTime, 1, 10000)
	// 实现导出 ...

	return nil, nil
}
