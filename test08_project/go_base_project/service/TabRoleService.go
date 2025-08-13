package service

// TabRoleService 角色 Service 层
type TabRoleService struct {
}

// InsertTabRole 新增角色
func (this *TabRoleService) InsertTabRole(tabRole *model.TabRole) (res any, err error) {

	return tabRole.InsertTabRole(core.GLOBAL_DB)
}

// UpdateTabRole 修改角色
func (this *TabRoleService) UpdateTabRole(tabRole *model.TabRole) (res any, err error) {

	return tabRole.UpdateTabRoleById(core.GLOBAL_DB)
}

// DeleteTabRole 删除角色
func (this *TabRoleService) DeleteTabRole(idList []int64) (res any, err error) {

	return (&model.TabRole{}).DeleteTabRole(core.GLOBAL_DB, idList)
}

// GetTabRoleById 获取角色业务详细信息
func (this *TabRoleService) GetTabRoleById(id int64) (res any, err error) {

	tabRole := model.TabRole{}
	err = (&tabRole).FindTabRoleById(core.GLOBAL_DB, id)
	if err != nil {
		return nil, err
	}

	return tabRole, nil
}

// GetTabRoleList 查询角色业务列表
func (this *TabRoleService) GetTabRoleList(tabRoleDto *dto.TabRoleDto) (res any, err error) {

	tabRole, err := tabRoleDto.DtoToModel()
	tabRoleList, err := tabRole.FindTabRoleList(core.GLOBAL_DB, tabRoleDto.SatrtTime, tabRoleDto.EndTime)
	if err != nil {
		return nil, err
	}

	return tabRoleList, nil
}

// GetTabRolePageList 分页查询角色业务列表
func (this *TabRoleService) GetTabRolePageList(tabRoleDto *dto.TabRoleDto) (res any, err error) {

	tabRole, err := tabRoleDto.DtoToModel()
	tabRoleList, total, err := tabRole.FindTabRolePageList(core.GLOBAL_DB, tabRoleDto.SatrtTime, tabRoleDto.EndTime, tabRoleDto.PageNum, tabRoleDto.PageSize)
	if err != nil {
		return nil, err
	}

	return commonUtils.BuildPageData[model.TabRole](tabRoleList, total, tabRoleDto.PageNum, tabRoleDto.PageSize), nil
}

// ExportTabRole 导出角色业务列表
func (this *TabRoleService) ExportTabRole(tabRoleDto *dto.TabRoleDto) (res any, err error) {

	tabRole, err := tabRoleDto.DtoToModel()
	tabRole.FindTabRolePageList(core.GLOBAL_DB, tabRoleDto.SatrtTime, tabRoleDto.EndTime, 1, 10000)
	// 实现导出 ...

	return nil, nil
}
