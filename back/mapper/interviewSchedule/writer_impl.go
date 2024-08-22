package interviewSchedule

import (
	hp "evanBlog/common/dehelper"
	"evanBlog/config/mysql"
)

type writerImpl struct {
	hp *hp.DbHelper
}

func newWriterImpl() *writerImpl {
	return &writerImpl{
		hp: mysql.DBHelper,
	}
}
