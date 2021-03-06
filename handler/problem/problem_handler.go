package problem

import (
	"encoding/json"
	. "experiment/handler"
	"experiment/model"
	"experiment/pkg/errno"
	"experiment/pkg/message"
	"experiment/util"
	"github.com/gin-gonic/gin"
	"github.com/lexkong/log"
	"strconv"
)

func Create(c *gin.Context) {
	var request UploadRequest
	if err := c.Bind(&request); err != nil {
		SendResponse(c, errno.ErrBind, nil)
		log.Error("", err)
		return
	}

	example, err := json.Marshal(request.Example)
	if err != nil {
		SendResponse(c, errno.ErrJsonMarshal, nil)
		return
	}
	output, err := json.Marshal(request.Output)
	if err != nil {
		SendResponse(c, errno.ErrJsonMarshal, nil)
		return
	}

	username, _ := c.Get("username")
	p := model.ProblemModel{
		Title:       request.Title,
		Description: request.Description,
		Example:     example,
		Solution:    request.Solution,
		Output:      output,
		Poster:      username.(string),
	}

	if err := p.Validate(); err != nil {
		SendResponse(c, errno.ErrValidation, nil)
		return
	}

	if err := p.Create(); err != nil {
		SendResponse(c, errno.ErrDatabase, nil)
		return
	}

	SendResponse(c, errno.OK, map[string]int{
		"problemId": p.ProblemId,
	})
}

func UploadData(c *gin.Context) {
	problemId := c.Param("id")
	if len(problemId) == 0 {
		SendResponse(c, errno.ErrFileInit, nil)
		return
	}

	file, _, err := c.Request.FormFile("data")
	if err != nil {
		SendResponse(c, errno.ErrFileInit, nil)
		return
	}

	fileName, data, err := util.StoreFile(file)
	if err != nil {
		SendResponse(c, errno.ErrFileInit, nil)
		return
	}

	p := &model.ProblemModel{}
	if err := p.Update(problemId, map[string]interface{}{"data": fileName}); err != nil {
		SendResponse(c, errno.ErrDatabase, nil)
		return
	}

	// 发送至 MQ
	msg := message.TopicProblemMessage{
		ProblemId:  p.ProblemId,
		DataSource: data,
		Solution:   p.Solution,
		OutPut:     p.Output,
	}

	realMsg, err := util.MsgEncode(msg)
	if err != nil {
		SendResponse(c, errno.ErrJsonMarshal, nil)
		return
	}

	client := message.GetKafkaClient()
	err = client.Produce(message.TopicProblem, realMsg)
	if err != nil {
		// 发送 mq 失败
		SendResponse(c, errno.ErrSendMsgFail, nil)
		return
	}

	SendResponse(c, errno.OK, nil)
}

func Detail(c *gin.Context) {
	id := c.Param("id")
	if len(id) == 0 {
		SendResponse(c, errno.ErrParam, nil)
		return
	}

	p := model.ProblemModel{}
	if err := p.Detail(id); err != nil {
		SendResponse(c, errno.ErrDatabase, nil)
		return
	}

	example := model.Example{}
	err := json.Unmarshal(p.Example, &example)
	if err != nil {
		SendResponse(c, errno.ErrJsonUnmarshal, nil)
		return
	}
	output := model.Output{}
	err = json.Unmarshal(p.Output, &output)
	if err != nil {
		SendResponse(c, errno.ErrJsonUnmarshal, nil)
		return
	}

	SendResponse(c, errno.OK, DetailResponse{
		ProblemId:   p.ProblemId,
		Title:       p.Title,
		Description: p.Description,
		Example:     example,
		Output:      output,
		Poster:      p.Poster,
	})

}

func List(c *gin.Context) {
	indexSrc, limitSrc := c.Query("index"), c.Query("limit")
	index, err := strconv.Atoi(indexSrc)
	limit, err := strconv.Atoi(limitSrc)
	if err != nil {
		SendResponse(c, errno.ErrParam, nil)
		return
	}
	p := &model.ProblemModel{}
	list, err := p.List(index, limit)
	if err != nil {
		SendResponse(c, errno.ErrDatabase, nil)
		return
	}
	SendResponse(c, errno.OK, list)
}

func GetTotal(c *gin.Context) {
	p := &model.ProblemModel{}
	count, err := p.Total()
	if err != nil {
		SendResponse(c, errno.ErrDatabase, nil)
		return
	}
	SendResponse(c, errno.OK, map[string]interface{}{
		"count": count,
	})
}
