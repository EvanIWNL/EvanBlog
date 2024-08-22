package interview

import (
	"context"
	"evanBlog/mapper/interviewSchedule"
)

type InterviewLogic interface {
	GetInterviewScheduleListByPage(ctx context.Context, page int, pageSize int)
}

func NewInterviewLogic() InterviewLogic {
	return newInterviewLogicImpl()
}

type interviewLogicImpl struct {
	interviewScheduleR interviewSchedule.Reader
	interviewScheduleW interviewSchedule.Writer
}

func newInterviewLogicImpl() *interviewLogicImpl {
	return &interviewLogicImpl{
		interviewScheduleR: interviewSchedule.NewReader(),
		interviewScheduleW: interviewSchedule.NewWriter(),
	}
}
