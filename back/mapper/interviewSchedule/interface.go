package interviewSchedule

import (
	"context"
	"evanBlog/entity/tableobj"
)

type Reader interface {
	GetScheduleListByPage(ctx context.Context, limit int, offset int) (res []*tableobj.TInterviewSchedule, err error)
}

func NewReader() Reader {
	return newReaderImpl()
}

type Writer interface {
}

func NewWriter() Writer {
	return newWriterImpl()
}
