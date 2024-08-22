package interviewSchedule

import (
	"context"
	hp "evanBlog/common/dehelper"
	"evanBlog/config/mysql"
	"evanBlog/entity/rspcode"
	"evanBlog/entity/tableobj"
)

type readerImpl struct {
	hp *hp.DbHelper
}

func newReaderImpl() *readerImpl {
	return &readerImpl{
		hp: mysql.DBHelper,
	}
}

func (r *readerImpl) GetInterviewScheduleListByPage(ctx context.Context, limit int, offset int) (
	res []*tableobj.TInterviewSchedule, err error) {
	var data []*tableobj.TInterviewSchedule
	query := r.hp.GetDb(ctx).Limit(limit).Offset(offset).Find(&data)
	if query.Error != nil {
		return nil, rspcode.NewError(rspcode.CodeDatabaseError, "面试时间表查询失败")
	}
	if len(data) == 0 {
		return nil, nil
	}
	return data, nil
}
